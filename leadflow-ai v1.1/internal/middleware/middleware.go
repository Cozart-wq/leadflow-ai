// Package middleware содержит сквозную HTTP-логику (логирование, recovery),
// которая должна применяться ко всем или почти всем маршрутам.
//
// Почему middleware — отдельный пакет, а не часть internal/api:
//   - middleware не знает о конкретных маршрутах и бизнес-логике,
//     это горизонтальный слой (cross-cutting concern), в отличие от
//     api, который знает о конкретных ресурсах (leads, tasks и т.д.);
//   - такое разделение позволяет переиспользовать middleware в будущем
//     для разных роутеров, не таща за собой зависимости от api.
package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseWriter оборачивает http.ResponseWriter, чтобы перехватить
// код статуса ответа. Стандартный http.ResponseWriter не даёт способа
// узнать, какой статус был отправлен, — WriteHeader() вызывается один
// раз и результат никуда не сохраняется.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// RequestLogger логирует метод, путь, код ответа и время обработки
// каждого запроса. Это базовая наблюдаемость (observability), без которой
// невозможно понять, что происходит в production-сервисе.
func RequestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// По умолчанию 200, так как WriteHeader может быть не вызван
			// явно (net/http сам отправляет 200, если обработчик просто
			// пишет тело ответа).
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			log.Info("http_request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", rw.statusCode),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

// Recoverer перехватывает панику в обработчиках, логирует её и возвращает
// клиенту 500 вместо падения всего процесса. Без этого middleware одна
// необработанная паника в любом обработчике уронит весь сервер, включая
// все остальные активные запросы.
func Recoverer(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic_recovered",
						zap.Any("error", rec),
						zap.String("path", r.URL.Path),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
