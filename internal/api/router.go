package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/models"
)

func NewRouter(log *slog.Logger, subscriptions SubscriptionStorage) http.Handler {
	handler := NewSubscriptionHandler(log, subscriptions)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(requestLogger(log))
	router.Use(middleware.Recoverer)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Get("/swagger", swaggerUI)
	router.Get("/swagger/openapi.yaml", swaggerSpec)

	router.Route("/subscriptions", func(r chi.Router) {
		r.With(validateBody[models.SubscriptionRequest]).Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/summary", handler.Summary)
		r.Get("/{id}", handler.GetByID)
		r.With(validateBody[models.SubscriptionRequest]).Put("/{id}", handler.Update)
		r.Delete("/{id}", handler.Delete)
	})

	return router
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			log.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration", time.Since(start).String(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
