# Makefile документирует и стандартизирует часто используемые команды.
# Цель — чтобы любой разработчик мог посмотреть `make help` вместо
# поиска нужной команды в README или истории терминала.

MIGRATE_DSN=postgres://leadflow:leadflow@localhost:5432/leadflow?sslmode=disable

.PHONY: help run build test lint tidy migrate-up migrate-down migrate-create docker-up docker-down docker-logs

help: ## Показать список доступных команд
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

run: ## Запустить сервер локально (без Docker)
	go run ./cmd/server

build: ## Собрать бинарник в ./bin/server
	go build -o ./bin/server ./cmd/server

test: ## Запустить все тесты
	go test ./... -v

lint: ## Запустить статический анализ (требует golangci-lint)
	golangci-lint run ./...

tidy: ## Привести go.mod/go.sum в соответствие с кодом
	go mod tidy

migrate-up: ## Применить все миграции к БД
	migrate -database "$(MIGRATE_DSN)" -path migrations up

migrate-down: ## Откатить последнюю миграцию
	migrate -database "$(MIGRATE_DSN)" -path migrations down 1

migrate-create: ## Создать новую миграцию: make migrate-create name=add_tasks_table
	migrate create -ext sql -dir migrations -seq $(name)

docker-up: ## Поднять весь стек (app + postgres) в Docker
	docker compose up --build

docker-down: ## Остановить и удалить контейнеры
	docker compose down

docker-logs: ## Показать логи приложения
	docker compose logs -f app
