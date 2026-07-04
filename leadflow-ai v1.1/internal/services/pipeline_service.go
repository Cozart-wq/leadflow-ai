package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/leadflow-ai/leadflow-ai/internal/agents/analyzer"
	"github.com/leadflow-ai/leadflow-ai/internal/agents/contacts"
	"github.com/leadflow-ai/leadflow-ai/internal/agents/crawler"
	"github.com/leadflow-ai/leadflow-ai/internal/agents/outreach"
	"github.com/leadflow-ai/leadflow-ai/internal/agents/search"
	"github.com/leadflow-ai/leadflow-ai/internal/ai"
	"github.com/leadflow-ai/leadflow-ai/internal/models"
	"github.com/leadflow-ai/leadflow-ai/internal/repositories"
)

// PipelineService оркестрирует всех агентов: Search -> Crawler -> Contacts
// -> Analyzer -> Outreach, и сохраняет результат как Lead. Каждый агент
// остаётся независимым (не знает о других агентах) — единственное место,
// которое знает про весь пайплайн целиком, это сам PipelineService.
type PipelineService struct {
	search   search.Agent
	crawler  *crawler.Agent
	contacts *contacts.Agent
	analyzer *analyzer.Agent
	outreach *outreach.Agent

	leadRepo *repositories.LeadRepository
	tasks    *TaskService

	log *zap.Logger
}

func NewPipelineService(
	searchAgent search.Agent,
	crawlerAgent *crawler.Agent,
	contactsAgent *contacts.Agent,
	analyzerAgent *analyzer.Agent,
	outreachAgent *outreach.Agent,
	leadRepo *repositories.LeadRepository,
	tasks *TaskService,
	log *zap.Logger,
) *PipelineService {
	return &PipelineService{
		search:   searchAgent,
		crawler:  crawlerAgent,
		contacts: contactsAgent,
		analyzer: analyzerAgent,
		outreach: outreachAgent,
		leadRepo: leadRepo,
		tasks:    tasks,
		log:      log,
	}
}

const maxCompaniesPerTask = 10

// RunAsync запускает пайплайн в отдельной горутине. HTTP-обработчик не
// ждёт завершения (поиск + краулинг + AI-анализ нескольких сайтов может
// занять десятки секунд) — прогресс отслеживается через статус задачи.
func (s *PipelineService) RunAsync(taskID uuid.UUID, query string) {
	go func() {
		ctx := context.Background()
		if err := s.run(ctx, taskID, query); err != nil {
			s.log.Error("pipeline failed", zap.String("task_id", taskID.String()), zap.Error(err))
			errMsg := err.Error()
			_ = s.tasks.UpdateStatus(ctx, taskID, models.TaskStatusFailed, &errMsg)
		}
	}()
}

func (s *PipelineService) run(ctx context.Context, taskID uuid.UUID, query string) error {
	if err := s.tasks.UpdateStatus(ctx, taskID, models.TaskStatusRunning, nil); err != nil {
		return fmt.Errorf("pipeline: update status running: %w", err)
	}

	companies, err := s.search.Find(ctx, query, maxCompaniesPerTask)
	if err != nil {
		return fmt.Errorf("pipeline: search agent: %w", err)
	}

	for _, company := range companies {
		if err := s.processCompany(ctx, taskID, company); err != nil {
			// Ошибка на одной компании не должна прерывать всю задачу —
			// логируем и переходим к следующей.
			s.log.Warn("failed to process company",
				zap.String("company", company.Name),
				zap.String("website", company.Website),
				zap.Error(err))
			continue
		}
	}

	return s.tasks.UpdateStatus(ctx, taskID, models.TaskStatusCompleted, nil)
}

func (s *PipelineService) processCompany(ctx context.Context, taskID uuid.UUID, company search.Company) error {
	exists, err := s.leadRepo.ExistsByWebsite(ctx, company.Website)
	if err != nil {
		return fmt.Errorf("check existing lead: %w", err)
	}
	if exists {
		return nil
	}

	html, err := s.crawler.Fetch(ctx, company.Website)
	if err != nil {
		return fmt.Errorf("crawl %s: %w", company.Website, err)
	}

	contactInfo := s.contacts.Extract(html)
	pageText := analyzer.ExtractText(html)

	analysisResult, err := s.analyzer.Analyze(ctx, ai.AnalysisInput{
		CompanyName: company.Name,
		Website:     company.Website,
		Email:       contactInfo.Email,
		Phone:       contactInfo.Phone,
		Socials:     contactInfo.Socials,
		PageText:    pageText,
	})
	if err != nil {
		s.log.Warn("analyzer failed, saving lead without AI analysis", zap.Error(err))
	}

	var outreachMessage *string
	if analysisResult != nil {
		msg := s.outreach.PrepareMessage(analysisResult, company.Name)
		outreachMessage = &msg
	}

	lead := &models.Lead{
		ID:          uuid.New(),
		TaskID:      uuid.NullUUID{UUID: taskID, Valid: true},
		CompanyName: company.Name,
		Website:     strPtr(company.Website),
		Socials:     contactInfo.Socials,
		Status:      models.LeadStatusNew,
	}
	if contactInfo.Email != "" {
		lead.Email = strPtr(contactInfo.Email)
	}
	if contactInfo.Phone != "" {
		lead.Phone = strPtr(contactInfo.Phone)
	}
	if analysisResult != nil {
		score := analysisResult.QualityScore
		lead.QualityScore = &score
		lead.AIRecommendation = strPtr(analysisResult.Recommendation)
		lead.AIMessage = outreachMessage
		lead.Status = models.LeadStatusAnalyzed
	}

	if err := s.leadRepo.Create(ctx, lead); err != nil {
		return fmt.Errorf("save lead: %w", err)
	}

	return nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
