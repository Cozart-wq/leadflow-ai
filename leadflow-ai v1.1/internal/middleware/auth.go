package middleware

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/leadflow-ai/leadflow-ai/internal/auth"
)

type contextKey string

const claimsContextKey contextKey = "auth_claims"

// Auth проверяет Bearer-токен в заголовке Authorization и, если он валиден,
// кладёт распарсенные claims в контекст запроса. Middleware ничего не знает
// о базе данных пользователей — оно доверяет подписи токена (issuer тот же
// сервис, который его выпустил), что и делает JWT-аутентификацию
// stateless: не требуется поход в БД на каждый запрос ради проверки сессии.
func Auth(tokens *auth.TokenIssuer, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(header, "Bearer ")
			if !ok || token == "" {
				writeUnauthorized(w, "missing bearer token")
				return
			}

			claims, err := tokens.Parse(token)
			if err != nil {
				log.Debug("auth: token rejected", zap.Error(err))
				writeUnauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext извлекает claims аутентифицированного пользователя,
// ранее положенные туда middleware Auth. Второе возвращаемое значение
// false означает, что запрос прошёл через маршрут без Auth middleware —
// это ошибка сборки роутера, а не пользовательского ввода, поэтому
// обработчики, вызывающие эту функцию, вправе трактовать false как 401.
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*auth.Claims)
	return claims, ok
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
