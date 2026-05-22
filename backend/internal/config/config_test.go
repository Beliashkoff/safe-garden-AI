package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// allConfigEnvKeys lists every env var read by Config — used by tests to
// guarantee an isolated environment.
var allConfigEnvKeys = []string{
	"ENV", "HTTP_HOST", "HTTP_PORT", "LOG_LEVEL", "SENTRY_DSN",
	"POSTGRES_DSN",
	"JWT_KEYS_DIR", "JWT_ACTIVE_KID", "JWT_PRIVATE_KEY_PATH", "JWT_KID",
	"JWT_ACCESS_TTL", "REFRESH_TTL",
	"APPLE_BUNDLE_ID", "GOOGLE_CLIENT_ID_IOS", "GOOGLE_CLIENT_ID_ANDROID",
}

// isolateEnv unsets every config-related variable and restores the prior value
// via t.Setenv after the test.
func isolateEnv(t *testing.T) {
	t.Helper()
	for _, k := range allConfigEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			t.Setenv(k, v)
		}
		require.NoError(t, os.Unsetenv(k))
	}
}

func TestLoad_Defaults(t *testing.T) {
	isolateEnv(t)
	// POSTGRES_DSN is required at all envs; supply a placeholder.
	t.Setenv("POSTGRES_DSN", "postgres://dev:dev@localhost:5432/dev?sslmode=disable")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "dev", cfg.Env)
	assert.Equal(t, "", cfg.HTTPHost)
	assert.Equal(t, 8080, cfg.HTTPPort)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "", cfg.SentryDSN)
	assert.Equal(t, 15*time.Minute, cfg.JWTAccessTTL)
	assert.Equal(t, 720*time.Hour, cfg.RefreshTTL)
}

func TestLoad_Override(t *testing.T) {
	isolateEnv(t)
	// stage-equivalent: not prod, so validateProd does not run.
	t.Setenv("ENV", "dev")
	t.Setenv("HTTP_HOST", "127.0.0.1")
	t.Setenv("HTTP_PORT", "9000")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("SENTRY_DSN", "https://example@sentry.io/1")
	t.Setenv("POSTGRES_DSN", "postgres://user:pw@db:5432/sg?sslmode=disable")
	t.Setenv("JWT_ACCESS_TTL", "30m")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "dev", cfg.Env)
	assert.Equal(t, "127.0.0.1", cfg.HTTPHost)
	assert.Equal(t, 9000, cfg.HTTPPort)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "https://example@sentry.io/1", cfg.SentryDSN)
	assert.Equal(t, 30*time.Minute, cfg.JWTAccessTTL)
}

func TestLoad_InvalidPort(t *testing.T) {
	isolateEnv(t)
	t.Setenv("POSTGRES_DSN", "postgres://x")
	t.Setenv("HTTP_PORT", "not-a-number")

	_, err := Load()
	require.Error(t, err)
}

func TestLoad_MissingPostgresDSN(t *testing.T) {
	isolateEnv(t)

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "POSTGRES_DSN")
}

func TestLoad_Prod_RequiresOIDCAndJWT(t *testing.T) {
	isolateEnv(t)
	t.Setenv("ENV", "prod")
	t.Setenv("POSTGRES_DSN", "postgres://x")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APPLE_BUNDLE_ID")
	assert.Contains(t, err.Error(), "GOOGLE_CLIENT_ID")
	assert.Contains(t, err.Error(), "JWT_KEYS_DIR or JWT_PRIVATE_KEY_PATH")
}

func TestLoad_Prod_HappyPath_KeysDir(t *testing.T) {
	isolateEnv(t)
	t.Setenv("ENV", "prod")
	t.Setenv("POSTGRES_DSN", "postgres://x")
	t.Setenv("APPLE_BUNDLE_ID", "com.example.app")
	t.Setenv("GOOGLE_CLIENT_ID_IOS", "ios-client.apps.googleusercontent.com")
	t.Setenv("JWT_KEYS_DIR", "/secrets/jwt")
	t.Setenv("JWT_ACTIVE_KID", "2026-Q2")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "prod", cfg.Env)
}

func TestLoad_Prod_HappyPath_SingleKey(t *testing.T) {
	isolateEnv(t)
	t.Setenv("ENV", "prod")
	t.Setenv("POSTGRES_DSN", "postgres://x")
	t.Setenv("APPLE_BUNDLE_ID", "com.example.app")
	t.Setenv("GOOGLE_CLIENT_ID_ANDROID", "android-client.apps.googleusercontent.com")
	t.Setenv("JWT_PRIVATE_KEY_PATH", "/secrets/jwt.pem")
	t.Setenv("JWT_KID", "dev1")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "prod", cfg.Env)
}

func TestLoad_Prod_KeysDirMissingActiveKID(t *testing.T) {
	isolateEnv(t)
	t.Setenv("ENV", "prod")
	t.Setenv("POSTGRES_DSN", "postgres://x")
	t.Setenv("APPLE_BUNDLE_ID", "com.example.app")
	t.Setenv("GOOGLE_CLIENT_ID_IOS", "ios.apps.googleusercontent.com")
	t.Setenv("JWT_KEYS_DIR", "/secrets/jwt")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_ACTIVE_KID")
}
