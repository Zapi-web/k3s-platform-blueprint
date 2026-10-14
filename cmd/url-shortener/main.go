package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Zapi-web/url-shortener/internal/base62"
	"github.com/Zapi-web/url-shortener/internal/cache"
	"github.com/Zapi-web/url-shortener/internal/config"
	"github.com/Zapi-web/url-shortener/internal/database"
	"github.com/Zapi-web/url-shortener/internal/kgs"
	"github.com/Zapi-web/url-shortener/internal/logger"
	"github.com/Zapi-web/url-shortener/internal/metrics"
	"github.com/Zapi-web/url-shortener/internal/server"
	"github.com/Zapi-web/url-shortener/internal/service"
	"github.com/Zapi-web/url-shortener/internal/tracer"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Init()

	if err != nil {
		slog.Error("Failed to read config", "err", err)
		return 1
	}

	slog.SetDefault(logger.NewLogger(cfg.App.LogLevel))
	slog.Info("Logger initialized", "level", cfg.App.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	databaseInnerConfig := database.Config{
		MaxConns:          cfg.Database.MaxConns,
		MinConns:          cfg.Database.MinConns,
		MaxConnLifeTime:   cfg.Database.MaxConnLifeTime,
		MaxConnIdleTime:   cfg.Database.MaxConnIdleTime,
		HealthCheckPeriod: cfg.Database.HealthCheckPeriod,
		ConnectionTimeout: cfg.Database.ConnectionTimeout,
	}

	postgresDB, err := database.NewPostgres(ctx, cfg.Database.ConnString, databaseInnerConfig)
	if err != nil {
		slog.Error("failed to connect to a database", "err", err)
		return 1
	}
	defer postgresDB.Close()

	slog.Info("Database initialized")

	redisCache := cache.NewRedis(ctx, cfg.Cache.RedisAddrs, cfg.Cache.RedisMasterName, cfg.Cache.RedisPassword, cfg.Cache.CacheTTL)
	defer func() {
		if err := redisCache.Close(); err != nil {
			slog.Error("failed to close cache", "err", err)
		}
	}()

	appCache := service.Cache(redisCache)

	slog.Info("Cache initialized")

	kgs, err := kgs.New(cfg.Server.NodeID)
	if err != nil {
		slog.Error("failed to make an kgs", "err", err)
		return 1
	}

	encode := base62.New()

	var prodMetrics interface {
		service.Metrics
		server.Metrics
	}

	if cfg.App.MetricsEnable {
		prodMetrics = metrics.NewVM()
	} else {
		prodMetrics = metrics.NewFake()
	}

	if cfg.Tracer.TracesEnable && cfg.Tracer.CollectorAddr != "" && cfg.Tracer.Ratio > 0 {
		shutdown, err := tracer.New(ctx, cfg.App.AppName, cfg.App.Environment, cfg.Tracer.CollectorAddr, cfg.Tracer.Ratio)

		if err != nil {
			slog.Error("failed to initialize tracer, failover to work without tracer", "err", err)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdown(shutdownCtx); err != nil {
				slog.Error("failed to shutdown otel tracer")
			}
		}()
	}

	shortener := service.New(postgresDB, appCache, kgs, encode, prodMetrics, cfg.Database.DbTTL)
	handler := server.NewHandlers(shortener)
	mv := server.NewMiddleware(prodMetrics)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handler.ServeHealthz)
	mux.HandleFunc("POST /api/v1/", handler.ServeSaveUrl)
	mux.HandleFunc("GET /{id}", handler.ServeGetURL)
	if cfg.App.MetricsEnable {
		mux.HandleFunc("GET /metrics", metrics.ExposeMetrics)
	}

	var rootHandler http.Handler = mux
	if cfg.App.MetricsEnable {
		rootHandler = mv.MetricsMiddleware(mux)
	}

	httpServer := server.NewServer(cfg.Server.Port, rootHandler, cfg.Server.ReadTimeout, cfg.Server.WriteTimeout)
	serverError := make(chan error, 1)
	go func() {
		serverError <- httpServer.RunServer()
	}()

	select {
	case err := <-serverError:
		if err != nil {
			slog.Error("received an error from http server", "err", err)
			return 1
		}
	case <-ctx.Done():
		slog.Info("received a signal, starting graceful shutdown")

		srvShutdownCtx, srvCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer srvCancel()

		if err := httpServer.Shutdown(srvShutdownCtx); err != nil {
			slog.Error("failed to shutdown server", "err", err)
		}
	}

	slog.Debug("Server stopped")
	return 0
}
