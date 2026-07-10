package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"go.uber.org/zap"

	"warehouse06/internal/config"
	delivery "warehouse06/internal/delivery/http"
	"warehouse06/internal/parser"
	"warehouse06/internal/repository"
	"warehouse06/internal/staticfiles"
	"warehouse06/internal/sync"
)

var version = "dev"

func newLogger(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}

func main() {
	cfg := config.MustLoad()
	log, err := newLogger(cfg.Env)
	if err != nil {
		zap.L().Fatal("failed to initialize logger", zap.Error(err))
	}
	defer func() { _ = log.Sync() }()

	startupDSN := repository.ResolveStartupDSN(cfg.DSN)
	repo, err := repository.NewSQLiteRepository(startupDSN, log)
	if err != nil {
		log.Fatal("failed to initialize repository", zap.Error(err))
	}

	holder := repository.NewHolder(repo, startupDSN)
	defer func() {
		if active := holder.Get(); active != nil {
			_ = active.Close()
		}
	}()

	p := parser.NewParser(cfg.StorageDir, log)
	syncStatus := sync.NewStatus()
	syncer := sync.NewSyncer(
		cfg.StorageDir,
		cfg.StorageURL,
		cfg.DSN,
		cfg.SyncInterval(),
		holder,
		p,
		syncStatus,
		log,
	)

	syncCtx, cancelSync := context.WithCancel(context.Background())
	defer cancelSync()

	syncDone := make(chan struct{})
	go func() {
		defer close(syncDone)
		syncer.Run(syncCtx)
	}()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(delivery.RequestLogger(log))
	r.Use(middleware.Recoverer)
	// Server-side bound on request handling; DB queries inherit this ctx.
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(delivery.SecurityHeaders)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		ExposedHeaders:   []string{"Link", "Retry-After"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	h := delivery.NewHandler(holder, syncStatus, log)
	h.RegisterRoutes(r)

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	workDir, _ := os.Getwd()
	staticfiles.RegisterRoutes(r, staticfiles.RoutesConfig{
		StorageDir: cfg.StorageDir,
		WorkDir:    workDir,
	})

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Info("starting server", zap.Int("port", cfg.Port), zap.String("version", version))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down server...")

	cancelSync()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("server forced to shutdown", zap.Error(err))
	}

	// Wait for an in-flight sync (and its repo disposals) so we do not exit
	// mid-rebuild and leave a partially written staging database behind.
	select {
	case <-syncDone:
		syncer.Wait()
	case <-time.After(10 * time.Second):
		log.Warn("sync did not stop in time, exiting anyway")
	}

	log.Info("server exited properly")
}
