// Package config отвечает за загрузку и валидацию конфигурации приложения.
//
// Почему отдельный пакет, а не просто os.Getenv() в main.go:
//   - конфигурация используется во многих местах (database, server, logger),
//     и должна быть единой типизированной структурой, а не разрозненными
//     вызовами os.Getenv по всему коду;
//   - Viper даёт единый источник правды: значения могут приходить из
//     configs/config.yaml, переменных окружения или значений по умолчанию,
//     с чётким приоритетом (env переопределяет файл);
//   - структура Config документирует сама себя: видно, какие настройки
//     вообще существуют, без поиска по всему проекту.
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config — корневая структура конфигурации приложения.
// Каждый под-компонент (сервер, БД, логгер) получает свой вложенный конфиг,
// а не плоский список полей — это упрощает передачу конфигурации в
// конструкторы соответствующих пакетов (например, database.New(cfg.Database)).
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	AI       AIConfig       `mapstructure:"ai"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

// AuthConfig задаёт параметры выпуска JWT для аутентификации.
type AuthConfig struct {
	// JWTSecret подписывает access-токены (HMAC-SHA256). Обязателен
	// к переопределению в production через LEADFLOW_AUTH_JWT_SECRET —
	// значение по умолчанию годится только для локальной разработки
	// и явно логируется как небезопасное при старте (см. main.go).
	JWTSecret string `mapstructure:"jwt_secret"`

	// AccessTokenTTLMinutes — время жизни access-токена. Проект пока не
	// реализует refresh-токены (отдельный шаг v1.2+), поэтому TTL выбран
	// достаточно большим (24 часа) для приемлемого UX без повторных
	// логинов, но не бессрочным.
	AccessTokenTTLMinutes int `mapstructure:"access_token_ttl_minutes"`
}

// AIConfig задаёт, какой LLM-провайдер использовать для анализа лидов.
// Provider: openai | claude | gemini | mock (по умолчанию, без ключей API).
type AIConfig struct {
	Provider string `mapstructure:"provider"`
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
}

// ServerConfig содержит настройки HTTP-сервера.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`

	// Таймауты выставляются явно, а не берутся из умолчаний net/http,
	// потому что стандартный http.Server без таймаутов уязвим к
	// Slowloris-атакам и зависшим соединениям.
	ReadTimeoutSeconds  int `mapstructure:"read_timeout_seconds"`
	WriteTimeoutSeconds int `mapstructure:"write_timeout_seconds"`
	IdleTimeoutSeconds  int `mapstructure:"idle_timeout_seconds"`
}

// DatabaseConfig содержит настройки подключения к PostgreSQL.
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	Name            string `mapstructure:"name"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime_minutes"`
}

// DSN собирает строку подключения к PostgreSQL.
// Метод живёт на DatabaseConfig, а не в пакете database, потому что
// это чистое преобразование данных конфигурации — ему не нужен доступ
// к *sql.DB или сети.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// LoggerConfig содержит настройки логирования.
type LoggerConfig struct {
	// Level: debug | info | warn | error
	Level string `mapstructure:"level"`
	// Format: json (production) | console (локальная разработка)
	Format string `mapstructure:"format"`
}

// Load читает конфигурацию из файла configs/config.yaml и переменных окружения.
//
// Приоритет источников (от низшего к высшему):
//  1. значения по умолчанию (setDefaults);
//  2. файл configs/config.yaml;
//  3. переменные окружения с префиксом LEADFLOW_.
//
// Такой порядок — стандартная практика 12-factor app: файл задаёт базовую
// конфигурацию для разработки, а окружение (Docker, CI/CD, продакшен)
// переопределяет чувствительные или окружение-специфичные значения
// (пароли, хосты) без изменения файлов в репозитории.
func Load(path string) (*Config, error) {
	v := viper.New()

	setDefaults(v)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(path)

	// Переменные окружения вида LEADFLOW_DATABASE_HOST переопределяют
	// database.host из yaml. Точки заменяются на подчёркивания, так как
	// переменные окружения не поддерживают точки в именах.
	v.SetEnvPrefix("leadflow")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: failed to read config file: %w", err)
		}
		// Отсутствие файла — не фатальная ошибка: приложение может работать
		// полностью на переменных окружения (типичный случай в контейнере).
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: failed to unmarshal config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config: invalid config: %w", err)
	}

	return &cfg, nil
}

// IsDefaultJWTSecret возвращает true, если секрет не был переопределён
// и используется небезопасное значение по умолчанию для разработки.
// main.go использует это, чтобы явно предупредить в логах при старте,
// а не проваливать запуск — блокировать локальную разработку без причины
// было бы избыточно, но production-развёртывание без переопределения
// секрета должно быть заметно в логах с первой же строки.
func (c *Config) IsDefaultJWTSecret() bool {
	return c.Auth.JWTSecret == "dev-only-insecure-secret-change-me-in-production"
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout_seconds", 10)
	v.SetDefault("server.write_timeout_seconds", 10)
	v.SetDefault("server.idle_timeout_seconds", 60)

	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "leadflow")
	v.SetDefault("database.password", "leadflow")
	v.SetDefault("database.name", "leadflow")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 25)
	v.SetDefault("database.conn_max_lifetime_minutes", 5)

	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")

	v.SetDefault("ai.provider", "mock")

	// Значение по умолчанию подходит только для локальной разработки:
	// оно одинаково у всех, кто склонировал репозиторий, и не должно
	// использоваться в production. cfg.validate() лишь проверяет длину
	// (минимум для HMAC-SHA256), а предупреждение о значении по умолчанию
	// печатается в main.go, где уже доступен логгер.
	v.SetDefault("auth.jwt_secret", "dev-only-insecure-secret-change-me-in-production")
	v.SetDefault("auth.access_token_ttl_minutes", 1440)
}

// validate проверяет конфигурацию на очевидные ошибки ещё до старта
// приложения. Цель — упасть сразу с понятной ошибкой, а не через минуту
// работы с непонятным паническим стектрейсом где-то в глубине кода.
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database.host must not be empty")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database.name must not be empty")
	}

	switch c.Logger.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logger.level must be one of debug|info|warn|error, got %q", c.Logger.Level)
	}

	switch c.Logger.Format {
	case "json", "console":
	default:
		return fmt.Errorf("logger.format must be one of json|console, got %q", c.Logger.Format)
	}

	switch c.AI.Provider {
	case "openai", "claude", "anthropic", "gemini", "mock":
	default:
		return fmt.Errorf("ai.provider must be one of openai|claude|gemini|mock, got %q", c.AI.Provider)
	}

	if len(c.Auth.JWTSecret) < 32 {
		return fmt.Errorf("auth.jwt_secret must be at least 32 characters")
	}
	if c.Auth.AccessTokenTTLMinutes <= 0 {
		return fmt.Errorf("auth.access_token_ttl_minutes must be positive")
	}

	return nil
}
