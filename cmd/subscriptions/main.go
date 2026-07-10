package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/config"
	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/logger"
	"github.com/damakalshchikov/test-task-junior-golang-developer/internal/storage/postgres"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Env)
	log.Info("starting subscriptions service", "env", cfg.Env, "port", cfg.HTTP.Port)

	if err := postgres.RunMigrations(cfg.DB.MigrationsPath, cfg.DB.DSN(postgres.MigrateScheme)); err != nil {
		log.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	log.Info("migrations applied")

	pool, err := postgres.New(context.Background(), cfg.DB.DSN("postgres"))
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	log.Info("connected to database", "host", cfg.DB.Host, "database", cfg.DB.Name)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Recoverer)

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	log.Info("server started", "addr", server.Addr)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("server shutdown error", "error", err)
	}

	log.Info("server stopped")
}
