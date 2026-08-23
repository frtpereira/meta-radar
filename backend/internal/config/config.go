package config

import "os"

// Config holds everything the app needs from its environment. Keeping it in
// one place makes it obvious what has to be set for local dev vs docker vs
// prod, and avoids os.Getenv calls scattered across the codebase.
type Config struct {
	Port             string
	DatabaseURL      string
	LimitlessAPIBase string
	LimitlessAPIKey  string
	WebhookSecret    string
	TLSCertFile      string
	TLSKeyFile       string
}

func Load() Config {
	return Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://app:devpassword@localhost:5432/pokemontcg?sslmode=disable"),
		LimitlessAPIBase: getEnv("LIMITLESS_API_BASE", "https://play.limitlesstcg.com/api"),
		LimitlessAPIKey:  getEnv("LIMITLESS_API_KEY", ""),
		WebhookSecret:    getEnv("WEBHOOK_SECRET", ""),
		TLSCertFile:      getEnv("TLS_CERT_FILE", ""),
		TLSKeyFile:       getEnv("TLS_KEY_FILE", ""),
	}
}

// TLSEnabled reports whether both a certificate and key have been
// configured, meaning the API server should listen over HTTPS instead of
// plain HTTP.
func (c Config) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
