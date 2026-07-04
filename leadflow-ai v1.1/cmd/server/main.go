// cmd/server/main.go — точка входа приложения.
//
// Почему вся логика "склейки" (wiring) находится именно здесь, а не
// внутри пакетов config/logger/database:
//   - main.go — единственное место, которое имеет право знать обо ВСЕХ
//     пакетах приложения одновременно и решать порядок их инициализации.
//     Остальные пакеты (config, logger, database, api) ничего не знают
//     друг о друге напрямую — это и есть внедрение зависимостей (DI)
//     "руками", без DI-фреймворка, что достаточно для проекта этого размера;
//   - такой подход называют "explicit dependency injection" — все
//     зависимости передаются через конструкторы (New(...)), а не через
//     глобальные переменные или синглтоны, что упрощает тестирование
//     и делает граф зависимостей видимым в одном файле.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/leadflow-ai/leadflow-ai/internal/agents/analyzer"
	"github.com/leadflow-ai/leadflow-ai/internal/agents/contacts"
	"github.com/leadflow-ai/leadflow-ai/internal/agents/crawler"
	"github.com/leadflow-ai/leadflow-ai/internal/agents/outreach"
	"github.com/leadflow-ai/leadflow-ai/internal/agents/search"
	"github.com/leadflow-ai/leadflow-ai/internal/ai"
	"github.com/leadflow-ai/leadflow-ai/internal/api"
	"github.com/leadflow-ai/leadflow-ai/internal/auth"
	"github.com/leadflow-ai/leadflow-ai/internal/config"
	"github.com/leadflow-ai/leadflow-ai/internal/database"
	"github.com/leadflow-ai/leadflow-ai/internal/logger"
	"github.com/leadflow-ai/leadflow-ai/internal/repositories"
	"github.com/leadflow-ai/leadflow-ai/internal/services"
)

func main() {
	if err := run(); err != nil {
		// На этом этапе логгер может быть ещё не инициализирован
		// (например, ошибка в самой конфигурации), поэтому используется
		// os.Stderr напрямую как последний рубеж.
		os.Stderr.WriteString("fatal: " + err.Error() + "\n")
		os.Exit(1)
	}
}

// run содержит всю логику запуска и вынесен из main() для того, чтобы
// main() мог остаться простым и не смешивать "точку входа процесса"
// (os.Exit) с логикой инициализации, которую удобно тестировать.
func run() error {
	cfg, err := config.Load("configs")
	if err != nil {
		return err
	}

	log, err := logger.New(cfg.Logger)
	if err != nil {
		return err
	}
	defer log.Sync() //nolint:errcheck // Sync может вернуть ошибку на stdout/stderr в некоторых ОС — это не критично при завершении.

	log.Info("starting leadflow-ai",
		zap.String("version", "v1.1"),
	)

	if cfg.IsDefaultJWTSecret() {
		log.Warn("auth.jwt_secret is using the default development value; set LEADFLOW_AUTH_JWT_SECRET before deploying to production")
	}

	// Контекст, отменяемый при получении SIGINT/SIGTERM, используется для
	// graceful shutdown: даём приложению шанс корректно закрыть соединения
	// с БД и дождаться завершения активных HTTP-запросов вместо резкого
	// обрыва (что особенно важно в Docker/Kubernetes при перезапуске).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	log.Info("database connection established",
		zap.String("host", cfg.Database.Host),
		zap.Int("port", cfg.Database.Port),
		zap.String("name", cfg.Database.Name),
	)

	leadRepo := repositories.NewLeadRepository(db)
	taskRepo := repositories.NewTaskRepository(db)
	userRepo := repositories.NewUserRepository(db)

	tokenIssuer, err := auth.NewTokenIssuer(
		cfg.Auth.JWTSecret,
		time.Duration(cfg.Auth.AccessTokenTTLMinutes)*time.Minute,
	)
	if err != nil {
		return err
	}

	aiProvider, err := ai.New(cfg.AI)
	if err != nil {
		return err
	}
	log.Info("ai provider initialized", zap.String("provider", aiProvider.Name()))

	searchAgent := search.NewDuckDuckGoAgent()
	crawlerAgent := crawler.NewAgent()
	contactsAgent := contacts.NewAgent()
	analyzerAgent := analyzer.NewAgent(aiProvider)
	outreachAgent := outreach.NewAgent()

	authService := services.NewAuthService(userRepo, tokenIssuer)
	taskService := services.NewTaskService(taskRepo)
	leadService := services.NewLeadService(leadRepo)
	pipelineService := services.NewPipelineService(
		searchAgent, crawlerAgent, contactsAgent, analyzerAgent, outreachAgent,
		leadRepo, taskService, log,
	)

	authHandler := api.NewAuthHandler(authService, log)
	taskHandler := api.NewTaskHandler(taskService, pipelineService, log)
	leadHandler := api.NewLeadHandler(leadService, log)

	router := api.NewRouter(log, "web", tokenIssuer, authHandler, taskHandler, leadHandler) // ожидает web/html, web/css, web/js

	srv := &http.Server{
		Addr:         cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSeconds) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeoutSeconds) * time.Second,
	}

	// Сервер запускается в отдельной горутине, чтобы основная горутина
	// могла одновременно ждать сигнал завершения (ctx.Done()) ниже.
	serverErrCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	select {
	case err := <-serverErrCh:
		return err
	case <-ctx.Done():
		log.Info("shutdown signal received, shutting down gracefully")
	}

	// Даём активным запросам до 10 секунд на завершение перед принудительным
	// закрытием соединений.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Info("server stopped gracefully")
	return nil
}
