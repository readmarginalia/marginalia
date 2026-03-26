package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"marginalia/db"
	"marginalia/server"
)

func main() {
	token := os.Getenv("TOKEN")
	if token == "" {
		if b, err := os.ReadFile("/run/secrets/token"); err == nil {
			token = strings.TrimSpace(string(b))
		}
	}
	if token == "" {
		log.Fatal("TOKEN is required (env var or /run/secrets/token)")
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

	auth := server.AuthConfig{
		Token:              token,
		EnableRateLimit:    envBool("AUTH_RATE_LIMIT"),
		TrustProxy:         envBool("TRUST_PROXY"),
		RealIPHeaders:      envList("REAL_IP_HEADERS"),
		TrustedProxyRanges: mustParseTrustedProxyRanges(envList("TRUSTED_PROXIES")),
	}

	if auth.TrustProxy && len(auth.TrustedProxyRanges) == 0 {
		log.Println("WARNING: TRUST_PROXY is enabled but TRUSTED_PROXIES is empty — all peers are trusted to set client IP headers")
	}

	srv := server.New(database, auth, owner, theme)

	log.Printf("marginalia listening on :%s (rate_limit=%t trust_proxy=%t)", port, auth.EnableRateLimit, auth.TrustProxy)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
