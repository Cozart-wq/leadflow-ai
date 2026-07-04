// Package api собирает HTTP-роутер приложения: регистрирует middleware,
// health-check и (в будущих версиях) маршруты для лидов, задач и т.д.
//
// Почему Chi, а не net/http напрямую или тяжёлый фреймворк (gin, echo):
//   - net/http (до 1.22) не поддерживал path-параметры и группировку
//     маршрутов из коробки, что усложняет рост проекта;
//   - Chi — тонкая обёртка над net/http (полностью совместима с
//     http.Handler), не навязывает архитектуру, добавляет только
//     маршрутизацию и middleware-цепочки. Это соответствует принципу
//     "простые решения предпочтительнее сложных";
//   - gin/echo привносят собственные соглашения (Context вместо
//     http.ResponseWriter/Request), что усложняет переиспользование
//     стандартных библиотек и повышает порог входа без реальной выгоды
//     для проекта такого размера.
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/leadflow-ai/leadflow-ai/internal/auth"
	custommw "github.com/leadflow-ai/leadflow-ai/internal/middleware"
)

// Router собирает и настраивает HTTP-роутер приложения.
//
// webRoot — путь к корневой директории фронтенда, ожидается, что внутри
// неё лежат поддиректории html/, css/, js/ (см. структуру проекта в
// web/). Передаётся параметром, а не хардкодится, чтобы путь можно было
// переопределить в тестах или при другой раскладке файлов в контейнере.
func NewRouter(
	log *zap.Logger,
	webRoot string,
	tokens *auth.TokenIssuer,
	authHandler *AuthHandler,
	taskHandler *TaskHandler,
	leadHandler *LeadHandler,
) http.Handler {
	r := chi.NewRouter()

	// Порядок middleware важен: Recoverer должен быть первым, чтобы
	// перехватывать панику из всех последующих middleware и обработчиков,
	// включая сам RequestLogger.
	r.Use(custommw.Recoverer(log))
	r.Use(custommw.RequestLogger(log))

	// Health-check используется Docker Compose (healthcheck) и системами
	// оркестрации (Kubernetes readiness/liveness probes) для проверки,
	// что сервис жив и готов принимать трафик.
	r.Get("/health", handleHealth)

	// API-маршруты версии v1: auth (регистрация/вход), tasks (создание
	// задач поиска) и leads (просмотр/удаление сохранённых лидов).
	// Префикс /api/v1 закладывается сразу, чтобы будущие breaking changes
	// в API не ломали текущих клиентов.
	r.Route("/api/v1", func(r chi.Router) {
		// /auth/register и /auth/login должны оставаться доступными без
		// токена — иначе получить первый токен было бы неоткуда. Поэтому
		// они регистрируются вне защищённой группы, а /auth/me — внутри
		// неё, так как ему нужен уже аутентифицированный пользователь.
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)

			r.Group(func(r chi.Router) {
				r.Use(custommw.Auth(tokens, log))
				r.Get("/me", authHandler.Me)
			})
		})

		// tasks и leads принадлежат конкретному пользователю, поэтому вся
		// группа защищена middleware.Auth: без валидного токена обработчик
		// даже не будет вызван.
		r.Group(func(r chi.Router) {
			r.Use(custommw.Auth(tokens, log))

			r.Route("/tasks", func(r chi.Router) {
				r.Post("/", taskHandler.Create)
				r.Get("/", taskHandler.List)
				r.Get("/{id}", taskHandler.Get)
			})
			r.Route("/leads", func(r chi.Router) {
				r.Get("/", leadHandler.List)
				r.Get("/{id}", leadHandler.Get)
				r.Delete("/{id}", leadHandler.Delete)
			})
		})
	})

	// CSS и JS раздаются каждый из своей директории под соответствующим
	// префиксом — так web/css и web/js остаются отдельными, независимо
	// изменяемыми областями (что соответствует структуре проекта), а не
	// одной смешанной директорией статики.
	r.Handle("/css/*", http.StripPrefix("/css/", http.FileServer(http.Dir(webRoot+"/css"))))
	r.Handle("/js/*", http.StripPrefix("/js/", http.FileServer(http.Dir(webRoot+"/js"))))

	// HTML отдаётся последним и перехватывает всё остальное (в том числе
	// "/"), чтобы index.html открывался по корневому пути. Регистрируется
	// после /health и /api/v1, чтобы не перекрыть их.
	r.Handle("/*", http.FileServer(http.Dir(webRoot+"/html")))

	return r
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
