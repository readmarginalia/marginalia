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

	srv := server.New(database, token, owner, theme)

	log.Printf("marginalia listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
