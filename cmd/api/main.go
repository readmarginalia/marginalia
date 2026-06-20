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
	"marginalia/internal/telemetry/metrics"
	"marginalia/internal/telemetry/tracing"
)

func main() {
	appConfig, err := configuration.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	res, err := telemetry.BuildResource(ctx, appConfig.Environment)
	if err != nil {
		slog.Error("failed to build resource", "error", err)
		os.Exit(1)
	}

	logger, shutdownLogs, err := logging.CreateLogger(ctx, res, appConfig)
	if err != nil {
		slog.Error("failed to create logger", "error", err)
		os.Exit(1)
	}
	defer shutdownLogs(ctx)

	slog.SetDefault(logger)

	shutdownTracing, err := tracing.SetupTracing(ctx, res, appConfig)
	if err != nil {
		slog.Error("failed to setup tracing", "error", err)
		os.Exit(1)
	}
	defer shutdownTracing(ctx)

	shutdownMetrics, err := metrics.SetupMetrics(ctx, res, appConfig)
	if err != nil {
		slog.Error("failed to setup metrics", "error", err)
		os.Exit(1)
	}
	defer shutdownMetrics(ctx)

	theme, err := server.LoadTheme(appConfig.ThemeName)
	if err != nil {
		slog.Error("failed to load theme", "error", err)
		os.Exit(1)
	}

	database, err := db.Open(appConfig.DbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	if appConfig.TrustProxy && len(appConfig.TrustedProxyRanges) == 0 {
		slog.Warn("TRUST_PROXY is enabled but TRUSTED_PROXIES is empty — all peers are trusted to set client IP headers")
	}

	waybackClient, err := wayback.NewClient("https://web.archive.org", 60*time.Second, logger)
	if err != nil {
		slog.Error("failed to create wayback client", "error", err)
		os.Exit(1)
	}
	repository := recommendations.NewRepository(database)
	recommendationsService := recommendations.NewService(repository, waybackClient, logger)
	feedService := feed.NewService(recommendationsService, logger)

	app := &server.App{
		AppConfig:       appConfig,
		Database:        database,
		Owner:           appConfig.Owner,
		Theme:           theme,
		Feed:            feedService,
		Recommendations: recommendationsService,
	}

	midlewares := []func(http.Handler) http.Handler{
		tracing.AddTraceContext,
		correlation.AddCorrelationId,
		requestmeta.AddRequestMetadata(appConfig),
		logging.AddRequestLogging,
		mg_http.AddCors,
	}

	appHandler := server.New(app, midlewares...)

	slog.Info("marginalia listening",
		"port", appConfig.Port,
		"rate_limit", appConfig.AuthRateLimit,
		"trust_proxy", appConfig.TrustProxy)

	err = http.ListenAndServe(":"+appConfig.Port, appHandler)
	if err != nil {
		slog.Error("server stopped", "err", err, "port", appConfig.Port)
		os.Exit(1)
	}
}
