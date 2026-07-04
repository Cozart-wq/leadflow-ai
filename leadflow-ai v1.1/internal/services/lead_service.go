package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/leadflow-ai/leadflow-ai/internal/models"
	"github.com/leadflow-ai/leadflow-ai/internal/repositories"
)

type LeadService struct {
	leads *repositories.LeadRepository
}

func NewLeadService(leads *repositories.LeadRepository) *LeadService {
	return &LeadService{leads: leads}
}

func (s *LeadService) List(ctx context.Context, userID uuid.UUID, taskID *uuid.UUID, limit, offset int) ([]models.Lead, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.leads.List(ctx, userID, taskID, limit, offset)
}

func (s *LeadService) Get(ctx context.Context, userID, id uuid.UUID) (*models.Lead, error) {
	return s.leads.GetByID(ctx, userID, id)
}

func (s *LeadService) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return s.leads.Delete(ctx, userID, id)
}

func (s *LeadService) UpdateAnalysis(ctx context.Context, lead *models.Lead) error {
	if err := s.leads.Update(ctx, lead); err != nil {
		return fmt.Errorf("lead_service: update analysis: %w", err)
	}
	return nil
}
