package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	// envconfig не применяет default к пустой строке — нужно полностью убрать
	// переменные. Используем Unsetenv через t.Setenv с последующим Unsetenv:
	// в Go 1.21+ t.Setenv не имеет Unset, так что используем os.Unsetenv,
	// а Cleanup восстанавливает оригинальное состояние через t.Setenv для
	// каждой переменной.
	for _, k := range []string{"ENV", "HTTP_HOST", "HTTP_PORT", "LOG_LEVEL", "SENTRY_DSN"} {
		if v, ok := os.LookupEnv(k); ok {
			t.Setenv(k, v)
		}
		require.NoError(t, os.Unsetenv(k))
	}

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "dev", cfg.Env)
	assert.Equal(t, "", cfg.HTTPHost)
	assert.Equal(t, 8080, cfg.HTTPPort)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "", cfg.SentryDSN)
}

func TestLoad_Override(t *testing.T) {
	t.Setenv("ENV", "prod")
	t.Setenv("HTTP_HOST", "127.0.0.1")
	t.Setenv("HTTP_PORT", "9000")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SENTRY_DSN", "https://example@sentry.io/1")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "prod", cfg.Env)
	assert.Equal(t, "127.0.0.1", cfg.HTTPHost)
	assert.Equal(t, 9000, cfg.HTTPPort)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "https://example@sentry.io/1", cfg.SentryDSN)
}

func TestLoad_InvalidPort(t *testing.T) {
	t.Setenv("HTTP_PORT", "not-a-number")

	_, err := Load()
	require.Error(t, err)
}
