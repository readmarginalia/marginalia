package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"marginalia/internal/configuration"
	"marginalia/internal/correlation"
	"marginalia/internal/feed"
	"marginalia/internal/infra/db"
	mg_http "marginalia/internal/infra/http"
	"marginalia/internal/interop/wayback"
	"marginalia/internal/recommendations"
	"marginalia/internal/requestmeta"
	"marginalia/internal/server"
	"marginalia/internal/telemetry"
	"marginalia/internal/telemetry/logging"
	"marginalia/internal/telemetry/tracing"
	"marginalia/internal/worker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	appConfig, err := configuration.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		return err
	}

	ctx := context.Background()
	logger, shutdownTelemetry, err := telemetry.Init(ctx, appConfig)
	if err != nil {
		slog.Error("failed to initialize telemetry", "error", err)
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := shutdownTelemetry(shutdownCtx); err != nil {
			slog.Error("failed to shutdown telemetry", "error", err)
		}
	}()

	slog.SetDefault(logger)

	theme, err := server.LoadTheme(appConfig.ThemeName)
	if err != nil {
		slog.Error("failed to load theme", "error", err)
		return err
	}

	database, err := db.Open(appConfig.DbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		return err
	}
	defer database.Close()

	if appConfig.TrustProxy && len(appConfig.TrustedProxyRanges) == 0 {
		slog.Warn("TRUST_PROXY is enabled but TRUSTED_PROXIES is empty — all peers are trusted to set client IP headers")
	}

	workerPool := worker.NewWorkerPool(10, 100)
	workerPool.Start(ctx)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := workerPool.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown worker pool", "error", err)
		}
	}()

	waybackClient, err := wayback.NewClient("https://web.archive.org", 60*time.Second, logger)
	if err != nil {
		slog.Error("failed to create wayback client", "error", err)
		return err
	}
	repository := recommendations.NewRepository(database)
	recommendationsService := recommendations.NewService(repository, waybackClient, logger, workerPool)
	feedService := feed.NewService(recommendationsService, logger)

	app := &server.App{
		AppConfig:       appConfig,
		Database:        database,
		Owner:           appConfig.Owner,
		Theme:           theme,
		Feed:            feedService,
		Recommendations: recommendationsService,
	}

	middlewares := []func(http.Handler) http.Handler{
		tracing.AddTraceContext,
		correlation.AddCorrelationId,
		requestmeta.AddRequestMetadata(appConfig),
		logging.AddRequestLogging,
		mg_http.AddCors,
	}

	appHandler := server.New(app, middlewares...)

	slog.Info("marginalia listening",
		"port", appConfig.Port,
		"rate_limit", appConfig.AuthRateLimit,
		"trust_proxy", appConfig.TrustProxy)

	err = http.ListenAndServe(":"+appConfig.Port, appHandler)
	if err != nil {
		slog.Error("server stopped", "err", err, "port", appConfig.Port)
		return err
	}

	return nil
}
