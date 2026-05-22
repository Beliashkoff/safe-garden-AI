package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config aggregates runtime configuration loaded from environment variables.
//
// Postgres DSN is required at all envs because the API process cannot serve
// requests without a database. OIDC (Apple/Google) and JWT key locations are
// optional in dev to allow boot without external credentials, but validateProd
// enforces them when ENV=prod.
type Config struct {
	Env       string `envconfig:"ENV" default:"dev"`
	HTTPHost  string `envconfig:"HTTP_HOST" default:""`
	HTTPPort  int    `envconfig:"HTTP_PORT" default:"8080"`
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	SentryDSN string `envconfig:"SENTRY_DSN" default:""`

	PostgresDSN string `envconfig:"POSTGRES_DSN" required:"true"`

	// JWT — RS256 with kid rotation. Prefer JWT_KEYS_DIR (multi-key) for prod;
	// single-file fallback (JWT_PRIVATE_KEY_PATH + JWT_KID) is for dev convenience.
	JWTKeysDir        string        `envconfig:"JWT_KEYS_DIR" default:""`
	JWTActiveKID      string        `envconfig:"JWT_ACTIVE_KID" default:""`
	JWTPrivateKeyPath string        `envconfig:"JWT_PRIVATE_KEY_PATH" default:""`
	JWTKID            string        `envconfig:"JWT_KID" default:""`
	JWTAccessTTL      time.Duration `envconfig:"JWT_ACCESS_TTL" default:"15m"`
	RefreshTTL        time.Duration `envconfig:"REFRESH_TTL" default:"720h"`

	// OIDC — empty in dev, required in prod (see validateProd).
	AppleBundleID    string `envconfig:"APPLE_BUNDLE_ID" default:""`
	GoogleClientIOS  string `envconfig:"GOOGLE_CLIENT_ID_IOS" default:""`
	GoogleClientAndr string `envconfig:"GOOGLE_CLIENT_ID_ANDROID" default:""`
}

func Load() (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, fmt.Errorf("envconfig: %w", err)
	}
	if c.Env == "prod" {
		if err := c.validateProd(); err != nil {
			return nil, fmt.Errorf("prod config: %w", err)
		}
	}
	return &c, nil
}

func (c *Config) validateProd() error {
	var missing []string
	if c.AppleBundleID == "" {
		missing = append(missing, "APPLE_BUNDLE_ID")
	}
	if c.GoogleClientIOS == "" && c.GoogleClientAndr == "" {
		missing = append(missing, "GOOGLE_CLIENT_ID_IOS or GOOGLE_CLIENT_ID_ANDROID")
	}
	if c.JWTKeysDir == "" && c.JWTPrivateKeyPath == "" {
		missing = append(missing, "JWT_KEYS_DIR or JWT_PRIVATE_KEY_PATH")
	}
	if c.JWTKeysDir != "" && c.JWTActiveKID == "" {
		missing = append(missing, "JWT_ACTIVE_KID (required when JWT_KEYS_DIR is set)")
	}
	if c.JWTPrivateKeyPath != "" && c.JWTKID == "" {
		missing = append(missing, "JWT_KID (required when JWT_PRIVATE_KEY_PATH is set)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required vars: %s", strings.Join(missing, ", "))
	}
	return nil
}
