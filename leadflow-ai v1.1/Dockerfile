# Multi-stage build: первый этап собирает бинарник, второй — минимальный
# рантайм-образ. Так итоговый образ не содержит компилятор Go, исходники
# и промежуточные артефакты сборки — только сам бинарник и статику,
# что снижает размер образа (десятки МБ вместо ~1ГБ) и площадь атаки.

FROM golang:1.24-alpine AS builder

WORKDIR /build

# Сначала копируются только go.mod/go.sum и скачиваются зависимости —
# это отдельный слой Docker-кэша. Пока зависимости не меняются, Docker
# переиспользует этот слой и не скачивает их заново при каждой сборке,
# даже если поменялся код приложения.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 даёт статически слинкованный бинарник без зависимости
# от glibc/musl — он запустится в минимальном образе (alpine) без
# дополнительных системных библиотек.
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/bin/server ./cmd/server

FROM alpine:3.20

# ca-certificates нужен для TLS-запросов приложения наружу (в будущем —
# к AI-провайдерам вроде OpenAI/Claude API); tzdata — для корректной
# работы с часовыми поясами в логах и данных.
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/bin/server ./server
COPY --from=builder /build/configs ./configs
COPY --from=builder /build/web ./web
COPY --from=builder /build/migrations ./migrations

# Непривилегированный пользователь: если приложение будет скомпрометировано,
# у процесса не будет root-прав внутри контейнера.
RUN adduser -D -u 1000 leadflow
USER leadflow

EXPOSE 8080

ENTRYPOINT ["./server"]
