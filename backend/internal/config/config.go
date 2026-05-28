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
	// Web/server OAuth client. The mobile google_sign_in flow uses this as its
	// serverClientId, so the id_token's aud equals this value — it must be in
	// the verifier allowlist.
	GoogleClientWeb string `envconfig:"GOOGLE_CLIENT_ID_WEB" default:""`

	// SMTP — Mailer for OTP delivery. Dev defaults target the docker-compose
	// MailHog (localhost:1025, no auth, no TLS). Prod uses Yandex 360
	// (smtp.yandex.ru:465, implicit TLS, AUTH LOGIN) — see validateProd.
	SMTPHost     string `envconfig:"SMTP_HOST" default:"localhost"`
	SMTPPort     int    `envconfig:"SMTP_PORT" default:"1025"`
	SMTPUsername string `envconfig:"SMTP_USERNAME" default:""`
	SMTPPassword string `envconfig:"SMTP_PASSWORD" default:""`
	SMTPFrom     string `envconfig:"SMTP_FROM" default:"noreply@localhost"`
	SMTPFromName string `envconfig:"SMTP_FROM_NAME" default:"Safe Garden AI"`
	SMTPTLS      bool   `envconfig:"SMTP_TLS" default:"false"`

	// DocsEnabled serves the OpenAPI spec + Swagger UI at /v1/docs. Safe to
	// leave on in prod (the contract is not secret) but available to disable.
	DocsEnabled bool `envconfig:"DOCS_ENABLED" default:"true"`

	// RedisAddr — Managed Redis for per-user rate limiting (ARCH §8.2). Empty in
	// dev → message rate limiting is disabled (allow-all). Required in prod.
	RedisAddr     string `envconfig:"REDIS_ADDR" default:""`
	RedisPassword string `envconfig:"REDIS_PASSWORD" default:""`

	// UIDHashPepper — salt for uid_hash = sha256(user_id + pepper). The hash is
	// the only user identifier sent to the worker/Anthropic (ARCH §11.4, §8.6).
	// Required in prod.
	UIDHashPepper string `envconfig:"UID_HASH_PEPPER" default:""`
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
	require := func(value, name string) {
		if value == "" {
			missing = append(missing, name)
		}
	}

	require(c.AppleBundleID, "APPLE_BUNDLE_ID")
	require(c.SMTPUsername, "SMTP_USERNAME")
	require(c.SMTPPassword, "SMTP_PASSWORD")
	require(c.SMTPFrom, "SMTP_FROM")
	require(c.RedisAddr, "REDIS_ADDR")
	require(c.UIDHashPepper, "UID_HASH_PEPPER")

	if c.GoogleClientIOS == "" && c.GoogleClientAndr == "" && c.GoogleClientWeb == "" {
		missing = append(missing, "GOOGLE_CLIENT_ID_IOS or GOOGLE_CLIENT_ID_ANDROID or GOOGLE_CLIENT_ID_WEB")
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
