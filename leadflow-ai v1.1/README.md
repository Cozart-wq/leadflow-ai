# LeadFlow AI

AI-платформа для автоматизации лидогенерации: поиск компаний, сбор и анализ
информации о них, сохранение лидов, с последующим подключением AI-анализа
и автоматизации.

Текущая версия: **v1.1 — аутентификация и владение данными.**

## Стек

- **Backend:** Go 1.24, Chi Router, PostgreSQL, sqlx, Zap, Viper
- **Frontend:** HTML5, CSS3, Vanilla JS
- **AI:** OpenAI / Claude / Gemini / Mock — единый интерфейс `ai.Provider`
- **Auth:** JWT (HS256) + PBKDF2-HMAC-SHA256, обе реализации на стандартной библиотеке (`internal/auth`)
- **Инфраструктура:** Docker, Docker Compose

## Структура проекта

```
cmd/server/          точка входа приложения (main.go)
internal/
  api/                HTTP-роутер и обработчики (auth, tasks, leads)
  auth/               хэширование паролей и JWT (без внешних зависимостей)
  ai/                 абстракция AI-провайдера: OpenAI/Claude/Gemini/Mock
  config/             загрузка и валидация конфигурации (Viper)
  database/           подключение к PostgreSQL (sqlx)
  logger/             инициализация структурированного логгера (zap)
  middleware/         логирование запросов, recovery, JWT-аутентификация
  models/             User, Lead, Task
  repositories/       UserRepository, LeadRepository, TaskRepository
  services/           AuthService, LeadService, TaskService, PipelineService
  agents/
    search/            поиск компаний (DuckDuckGo HTML, без API-ключа)
    crawler/            загрузка HTML сайта
    contacts/           извлечение email/телефона/соцсетей
    analyzer/           AI-оценка качества лида
    outreach/           подготовка персонализированного сообщения
web/                  html/ css/ js/ — дашборд (с формой входа/регистрации)
migrations/            SQL-миграции (golang-migrate)
docs/decisions/        архитектурные решения (ADR)
```

## Как это работает

1. Пользователь регистрируется (`POST /api/v1/auth/register`) или входит
   (`POST /api/v1/auth/login`) и получает JWT.
2. С этим токеном (`Authorization: Bearer <token>`) пользователь создаёт
   задачу поиска через `POST /api/v1/tasks {"query": "..."}`.
3. `PipelineService` в фоне последовательно вызывает агентов:
   `Search → Crawler → Contacts → Analyzer (AI) → Outreach` для каждой найденной компании.
4. Результат сохраняется как `Lead` со скорингом, рекомендацией и готовым сообщением для контакта.
5. Прогресс отслеживается через `GET /api/v1/tasks/{id}` (`status`: pending → running → completed/failed).
6. Каждый пользователь видит только свои задачи и лиды — доступ к чужим данным по id невозможен.

## API

| Метод | Путь | Описание | Требует токен |
|---|---|---|---|
| POST | `/api/v1/auth/register` | Регистрация: `{"email","password","name"}` | нет |
| POST | `/api/v1/auth/login` | Вход: `{"email","password"}` | нет |
| GET | `/api/v1/auth/me` | Текущий пользователь | да |
| POST | `/api/v1/tasks` | Создать задачу поиска: `{"query": "..."}` | да |
| GET | `/api/v1/tasks` | Список своих задач | да |
| GET | `/api/v1/tasks/{id}` | Статус задачи | да |
| GET | `/api/v1/leads?task_id=&limit=&offset=` | Список своих лидов | да |
| GET | `/api/v1/leads/{id}` | Лид по ID | да |
| DELETE | `/api/v1/leads/{id}` | Удалить лид | да |
| GET | `/health` | Health-check | нет |

## Аутентификация

Все маршруты, кроме `/health`, `/api/v1/auth/register` и `/api/v1/auth/login`,
требуют JWT в заголовке `Authorization: Bearer <token>`. Токен выдаётся при
регистрации и входе и действителен `auth.access_token_ttl_minutes` минут
(по умолчанию 24 часа). Секрет подписи задаётся через
`LEADFLOW_AUTH_JWT_SECRET` — используемое по умолчанию значение подходит
только для локальной разработки, при старте с ним в лог пишется
предупреждение.

## AI-провайдеры

По умолчанию используется `mock` — эвристический анализ без внешних API,
проект работает "из коробки" без ключей. Чтобы подключить реальную модель:

```yaml
ai:
  provider: "openai"   # openai | claude | gemini | mock
  api_key: "sk-..."
  model: "gpt-4o-mini"  # необязательно, есть значения по умолчанию
```

или через переменные окружения: `LEADFLOW_AI_PROVIDER`, `LEADFLOW_AI_API_KEY`, `LEADFLOW_AI_MODEL`.

## Быстрый старт

### Через Docker Compose (рекомендуется)

```bash
docker compose up --build
```

Поднимет PostgreSQL и приложение. Приложение будет доступно на
`http://localhost:8080`, health-check — на `http://localhost:8080/health`.

### Локально (без Docker)

Требуется установленный Go 1.24+ и PostgreSQL.

```bash
# 1. Установить зависимости
go mod tidy

# 2. Поднять PostgreSQL (например, через Docker)
docker run -d --name leadflow-postgres \
  -e POSTGRES_USER=leadflow -e POSTGRES_PASSWORD=leadflow -e POSTGRES_DB=leadflow \
  -p 5432:5432 postgres:16-alpine

# 3. Применить миграции (требуется golang-migrate CLI:
#    https://github.com/golang-migrate/migrate)
make migrate-up

# 4. Запустить сервер
make run
```

Сервер запустится на `http://localhost:8080`, конфигурация берётся из
`configs/config.yaml` (см. `make help` для списка остальных команд).

## Конфигурация

Настройки задаются в `configs/config.yaml` и могут быть переопределены
переменными окружения с префиксом `LEADFLOW_`, например:

```bash
LEADFLOW_DATABASE_HOST=db.example.com
LEADFLOW_DATABASE_PASSWORD=secret
LEADFLOW_LOGGER_LEVEL=warn
```

## Версия

v1.1 — добавлена аутентификация (регистрация/вход по JWT) и владение
данными: каждый пользователь видит и может изменять только свои задачи и
лиды. v1.0 (этапы v0.1–v0.4) реализованы ранее: фундамент, работа с
лидами, анализ сайтов, AI-оценка и генерация сообщений. Архитектурное
решение по аутентификации описано в
[docs/decisions/0001-authentication.md](./docs/decisions/0001-authentication.md).

## Лицензия

MIT — см. [LICENSE](./LICENSE).
