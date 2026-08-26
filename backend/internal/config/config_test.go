package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadUsesEnvAndFallbacks(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("LIMITLESS_API_BASE", "https://limitless.test")
	t.Setenv("LIMITLESS_API_KEY", "api-key")
	t.Setenv("WEBHOOK_SECRET", "secret")

	cfg := Load()

	assert.Equal(t, "9090", cfg.Port)
	assert.Equal(t, "postgres://example", cfg.DatabaseURL)
	assert.Equal(t, "https://limitless.test", cfg.LimitlessAPIBase)
	assert.Equal(t, "api-key", cfg.LimitlessAPIKey)
	assert.Equal(t, "secret", cfg.WebhookSecret)
}

func TestLoadFallsBackForUnsetOrEmptyValues(t *testing.T) {
	for _, key := range []string{"PORT", "DATABASE_URL", "LIMITLESS_API_BASE", "LIMITLESS_API_KEY", "WEBHOOK_SECRET"} {
		_ = os.Unsetenv(key)
	}
	t.Setenv("PORT", "")
	cfg := Load()

	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "postgres://app:devpassword@localhost:5432/pokemontcg?sslmode=disable", cfg.DatabaseURL)
	assert.Equal(t, "https://play.limitlesstcg.com/api", cfg.LimitlessAPIBase)
	assert.Equal(t, "", cfg.LimitlessAPIKey)
	assert.Equal(t, "", cfg.WebhookSecret)
}

func TestGetEnv(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		t.Setenv("META_RADAR_TEST_KEY", "value")
		assert.Equal(t, "value", getEnv("META_RADAR_TEST_KEY", "fallback"))
	})

	t.Run("unset uses fallback", func(t *testing.T) {
		_ = os.Unsetenv("META_RADAR_TEST_KEY_UNSET")
		assert.Equal(t, "fallback", getEnv("META_RADAR_TEST_KEY_UNSET", "fallback"))
	})

	t.Run("empty uses fallback", func(t *testing.T) {
		t.Setenv("META_RADAR_TEST_KEY_EMPTY", "")
		assert.Equal(t, "fallback", getEnv("META_RADAR_TEST_KEY_EMPTY", "fallback"))
	})
}
