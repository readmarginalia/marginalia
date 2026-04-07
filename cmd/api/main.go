package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"marginalia/internal/auth"
	"marginalia/internal/common"
	"marginalia/internal/correlation"
	"marginalia/internal/feed"
	"marginalia/internal/identity"
	"marginalia/internal/infra/db"
	"marginalia/internal/interop/peerclient"
	"marginalia/internal/interop/wayback"
	"marginalia/internal/peers"
	"marginalia/internal/recommendations"
	"marginalia/internal/server"
	"marginalia/internal/telemetry"
	"marginalia/internal/telemetry/logging"
	"marginalia/internal/telemetry/tracing"
)

func main() {
	ctx := context.Background()
	res, err := telemetry.BuildResource()
	if err != nil {
		slog.Error("failed to build resource", "error", err)
		os.Exit(1)
	}

	logger, shutdownLogs, err := logging.CreateLogger(ctx, res)
	if err != nil {
		slog.Error("failed to create logger", "error", err)
		os.Exit(1)
	}
	defer shutdownLogs(ctx)

	slog.SetDefault(logger)

	shutdownTracing, err := tracing.SetupTracing(ctx, res)
	if err != nil {
		slog.Error("failed to setup tracing", "error", err)
		os.Exit(1)
	}
	defer shutdownTracing(ctx)

	token := os.Getenv("TOKEN")
	if token == "" {
		slog.Error("TOKEN is required")
		os.Exit(1)
	}

	owner := os.Getenv("OWNER")
	themeName := os.Getenv("THEME")

	theme, err := server.LoadTheme(themeName)
	if err != nil {
		slog.Error("failed to load theme", "error", err)
		os.Exit(1)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9595"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/marginalia.db"
	}

	database, err := db.Open(dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	auth := auth.AuthConfig{
		Token:              token,
		EnableRateLimit:    common.EnvBool("AUTH_RATE_LIMIT"),
		TrustProxy:         common.EnvBool("TRUST_PROXY"),
		RealIPHeaders:      common.EnvList("REAL_IP_HEADERS"),
		TrustedProxyRanges: common.MustParseTrustedProxyRanges(common.EnvList("TRUSTED_PROXIES")),
	}

	if auth.TrustProxy && len(auth.TrustedProxyRanges) == 0 {
		slog.Warn("TRUST_PROXY is enabled but TRUSTED_PROXIES is empty — all peers are trusted to set client IP headers")
	}

	waybackClient, err := wayback.NewClient("https://web.archive.org", 60*time.Second)
	if err != nil {
		slog.Error("failed to create wayback client", "error", err)
		os.Exit(1)
	}

	identityService := identity.NewService(identity.NewRepository(database))
	nodeIdentity, err := identityService.Bootstrap(ctx)
	if err != nil {
		slog.Error("failed to bootstrap identity", "error", err)
		os.Exit(1)
	}
	slog.Info("node identity ready", "public_key", nodeIdentity.EncodedPublicKey())

	repository := recommendations.NewRepository(database)
	recommendationsService := recommendations.NewService(repository, waybackClient)
	feedService := feed.NewService(recommendationsService)

	peerRepo := peers.NewRepository(database)
	peerService := peers.NewService(peerRepo, peerclient.New(10*time.Second))

	app := &server.App{
		AuthConfig:      &auth,
		Database:        database,
		Owner:           owner,
		Theme:           theme,
		Feed:            feedService,
		Recommendations: recommendationsService,
		Identity:        nodeIdentity,
		Peers:           peerService,
	}

	appHandler := tracing.AddTraceContext(
		correlation.AddCorrelationId(
			logging.AddRequestLogging(
				server.New(app),
			),
		),
	)

	slog.Info("marginalia listening",
		"port", port,
		"rate_limit", auth.EnableRateLimit,
		"trust_proxy", auth.TrustProxy)

	err = http.ListenAndServe(":"+port, appHandler)
	if err != nil {
		slog.Error("server stopped", "err", err, "port", port)
		os.Exit(1)
	}
}
