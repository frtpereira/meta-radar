package config

import "os"

// Config holds everything the app needs from its environment. Keeping it in
// one place makes it obvious what has to be set for local dev vs docker vs
// prod, and avoids os.Getenv calls scattered across the codebase.
type Config struct {
	Port              string
	DatabaseURL       string
	LimitlessAPIBase  string
	LimitlessAPIKey   string
	WebhookSecret     string
}

func Load() Config {
	return Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://app:devpassword@localhost:5432/pokemontcg?sslmode=disable"),
		LimitlessAPIBase: getEnv("LIMITLESS_API_BASE", "https://play.limitlesstcg.com/api"),
		LimitlessAPIKey:  getEnv("LIMITLESS_API_KEY", ""),
		WebhookSecret:    getEnv("WEBHOOK_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
