package config

import (
	"log/slog"
	"os"
)

type Config struct {
	Port   string
	DBPath string
	APIKey string
}

func Load(log *slog.Logger) Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "home-monitor.db"
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		log.Warn("API_KEY env var not set — all ingest requests will be rejected")
	}

	return Config{Port: port, DBPath: dbPath, APIKey: apiKey}
}
