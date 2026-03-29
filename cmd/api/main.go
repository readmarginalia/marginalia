package main

import (
	"log"
	"net/http"
	"os"

	"marginalia/internal/auth"
	"marginalia/internal/common"
	"marginalia/internal/feed"
	"marginalia/internal/infra/db"
	"marginalia/internal/recommendations"
	"marginalia/internal/server"
)

func main() {
	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN is required")
	}

	owner := os.Getenv("OWNER")
	themeName := os.Getenv("THEME")

	theme, err := server.LoadTheme(themeName)
	if err != nil {
		log.Fatalf("failed to load theme: %v", err)
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
		log.Fatalf("failed to open database: %v", err)
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
		log.Println("WARNING: TRUST_PROXY is enabled but TRUSTED_PROXIES is empty — all peers are trusted to set client IP headers")
	}

	repository := recommendations.NewRepository(database)
	recommendationsService := recommendations.NewService(repository)
	feedService := feed.NewService(recommendationsService)

	app := &server.App{
		AuthConfig:      &auth,
		Database:        database,
		Owner:           owner,
		Theme:           theme,
		Feed:            feedService,
		Recommendations: recommendationsService,
	}

	srv := server.New(app)

	log.Printf("marginalia listening on :%s (rate_limit=%t trust_proxy=%t)", port, auth.EnableRateLimit, auth.TrustProxy)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
