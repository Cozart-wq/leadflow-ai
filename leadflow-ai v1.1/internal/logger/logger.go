// Package logger инициализирует структурированный логгер (zap) на основе
// конфигурации приложения.
//
// Почему отдельный пакет, а не zap.NewProduction() напрямую в main.go:
//   - формат и уровень логирования зависят от конфигурации (dev/prod),
//     и эта логика должна быть в одном месте, а не размазана по коду;
//   - другие пакеты должны зависеть от интерфейса *zap.Logger,
//     переданного через конструктор (dependency injection), а не от
//     глобального логгера — это упрощает тестирование и делает
//     зависимости явными.
package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/leadflow-ai/leadflow-ai/internal/config"
)

// New создаёт *zap.Logger на основе LoggerConfig.
//
// Используется zap, а не стандартный log или альтернативы (logrus,
// slog), потому что:
//   - zap — один из самых быстрых структурированных логгеров в Go
//     экосистеме (важно для HTTP middleware, где логируется каждый запрос);
//   - структурированные (key-value) логи легко парсятся в продакшене
//     (ELK, Loki, Datadog и т.д.), в отличие от произвольного текста;
//   - slog (стандартная библиотека с Go 1.21+) — тоже валидный выбор,
//     но zap выбран за зрелую экосистему и привычные паттерны
//     (SugaredLogger, With()) для этого проекта.
func New(cfg config.LoggerConfig) (*zap.Logger, error) {
	level, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		return nil, fmt.Errorf("logger: invalid level %q: %w", cfg.Level, err)
	}

	var zapCfg zap.Config
	switch cfg.Format {
	case "json":
		// JSON-формат для продакшена: машиночитаемый, удобен для
		// систем сбора логов.
		zapCfg = zap.NewProductionConfig()
	case "console":
		// Человекочитаемый цветной формат для локальной разработки.
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	default:
		return nil, fmt.Errorf("logger: invalid format %q", cfg.Format)
	}

	zapCfg.Level = zap.NewAtomicLevelAt(level)
	// ISO8601 читабельнее для человека и однозначно парсится машиной,
	// в отличие от эпохи в секундах (значение по умолчанию у zap).
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	log, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("logger: failed to build: %w", err)
	}

	return log, nil
}
