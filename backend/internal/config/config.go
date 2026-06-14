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
// requests without a database. OAuth (Yandex ID / VK ID) and JWT key locations
// are optional in dev to allow boot without external credentials, but
// validateProd enforces them when ENV=prod.
type Config struct {
	Env       string `envconfig:"ENV" default:"dev"`
	HTTPHost  string `envconfig:"HTTP_HOST" default:""`
	HTTPPort  int    `envconfig:"HTTP_PORT" default:"8080"`
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	SentryDSN string `envconfig:"SENTRY_DSN" default:""`

	// Metrics — Prometheus /metrics on a dedicated plain-HTTP listener, scraped
	// over the internal docker network. Never published through Caddy or to the
	// public internet (ARCH §9). Empty host binds all interfaces inside the
	// container; the port is not mapped to the host.
	MetricsHost string `envconfig:"METRICS_HOST" default:""`
	MetricsPort int    `envconfig:"METRICS_PORT" default:"9100"`

	PostgresDSN string `envconfig:"POSTGRES_DSN" required:"true"`

	// JWT — RS256 with kid rotation. Prefer JWT_KEYS_DIR (multi-key) for prod;
	// single-file fallback (JWT_PRIVATE_KEY_PATH + JWT_KID) is for dev convenience.
	JWTKeysDir        string        `envconfig:"JWT_KEYS_DIR" default:""`
	JWTActiveKID      string        `envconfig:"JWT_ACTIVE_KID" default:""`
	JWTPrivateKeyPath string        `envconfig:"JWT_PRIVATE_KEY_PATH" default:""`
	JWTKID            string        `envconfig:"JWT_KID" default:""`
	JWTAccessTTL      time.Duration `envconfig:"JWT_ACCESS_TTL" default:"15m"`
	RefreshTTL        time.Duration `envconfig:"REFRESH_TTL" default:"720h"`

	// OAuth providers (Yandex ID / VK ID, 406-FZ-compliant RU sign-in) —
	// empty in dev, required in prod (see validateProd). The code exchange
	// runs server-side: secrets and PKCE verifiers never reach the client.
	YandexClientID     string `envconfig:"YANDEX_CLIENT_ID" default:""`
	YandexClientSecret string `envconfig:"YANDEX_CLIENT_SECRET" default:""`
	// Callback URI registered at oauth.yandex.ru; the mobile app intercepts it
	// in the system browser (custom scheme).
	YandexRedirectURI string `envconfig:"YANDEX_REDIRECT_URI" default:""`
	// VK ID numeric application ID. No secret: the mobile SDK is a public
	// client and the exchange is protected by PKCE (verifier held here).
	VKClientID string `envconfig:"VK_CLIENT_ID" default:""`
	// Deep link the VK ID SDK returns through (vk{client_id}://vk.ru); must
	// match the token-exchange redirect_uri parameter.
	VKRedirectURI string `envconfig:"VK_REDIRECT_URI" default:""`

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

	// Admin panel (/admin/v1, served via admin.<domain>). Enabled only when
	// ADMIN_EMAIL is set; it pins the single operator account — setup, sign-in
	// and password-reset codes are issued for this address and no other.
	// Deliberately NOT required in prod so an api deploy never crash-loops on a
	// missing var (the panel simply stays off until the env is updated).
	AdminEmail string `envconfig:"ADMIN_EMAIL" default:""`
	// Sliding session lifetime: activity extends the cookie/session expiry.
	AdminSessionTTL time.Duration `envconfig:"ADMIN_SESSION_TTL" default:"24h"`
	// Expected Origin for mutating admin requests (e.g.
	// https://admin.agronomai.site). Empty disables the check (dev).
	AdminAllowedOrigin string `envconfig:"ADMIN_ALLOWED_ORIGIN" default:""`

	// RedisAddr — Managed Redis for per-user rate limiting (ARCH §8.2). Empty in
	// dev → message rate limiting is disabled (allow-all). Required in prod.
	RedisAddr     string `envconfig:"REDIS_ADDR" default:""`
	RedisPassword string `envconfig:"REDIS_PASSWORD" default:""`

	// UIDHashPepper — salt for uid_hash = sha256(user_id + pepper). The hash is
	// the only user identifier sent to the worker/Anthropic (ARCH §11.4, §8.6).
	// Required in prod.
	UIDHashPepper string `envconfig:"UID_HASH_PEPPER" default:""`

	// Object Storage (S3-compatible): MinIO in dev, Yandex Object Storage in
	// prod. Used for presigned photo uploads (ARCH §4.3, §5). Required in prod.
	// Path-style addressing works for both MinIO and Yandex OS.
	S3Endpoint     string `envconfig:"S3_ENDPOINT" default:""`
	S3Region       string `envconfig:"S3_REGION" default:"ru-central1"`
	S3AccessKey    string `envconfig:"S3_ACCESS_KEY" default:""`
	S3SecretKey    string `envconfig:"S3_SECRET_KEY" default:""`
	S3Bucket       string `envconfig:"S3_BUCKET" default:""`
	S3UsePathStyle bool   `envconfig:"S3_USE_PATH_STYLE" default:"true"`

	// Internal API — the mTLS-only listener the llm-worker calls back into during
	// a tool-use turn (ARCH §11, recommend_fertilizer). Runs on a separate port
	// from the public API, never behind RequireAuth. In prod mTLS is mandatory
	// (the client cert is the authentication); in dev it serves plain HTTP on
	// localhost. Required mTLS vars enforced in validateProd.
	InternalHTTPHost         string `envconfig:"INTERNAL_HTTP_HOST" default:""`
	InternalHTTPPort         int    `envconfig:"INTERNAL_HTTP_PORT" default:"8090"`
	InternalMTLSEnabled      bool   `envconfig:"INTERNAL_MTLS_ENABLED" default:"false"`
	InternalMTLSCertPath     string `envconfig:"INTERNAL_MTLS_CERT_PATH" default:""`
	InternalMTLSKeyPath      string `envconfig:"INTERNAL_MTLS_KEY_PATH" default:""`
	InternalMTLSClientCAPath string `envconfig:"INTERNAL_MTLS_CLIENT_CA_PATH" default:""`
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

	require(c.YandexClientID, "YANDEX_CLIENT_ID")
	require(c.YandexClientSecret, "YANDEX_CLIENT_SECRET")
	require(c.YandexRedirectURI, "YANDEX_REDIRECT_URI")
	require(c.VKClientID, "VK_CLIENT_ID")
	require(c.VKRedirectURI, "VK_REDIRECT_URI")
	require(c.SMTPUsername, "SMTP_USERNAME")
	require(c.SMTPPassword, "SMTP_PASSWORD")
	require(c.SMTPFrom, "SMTP_FROM")
	require(c.RedisAddr, "REDIS_ADDR")
	require(c.UIDHashPepper, "UID_HASH_PEPPER")
	require(c.S3Endpoint, "S3_ENDPOINT")
	require(c.S3AccessKey, "S3_ACCESS_KEY")
	require(c.S3SecretKey, "S3_SECRET_KEY")
	require(c.S3Bucket, "S3_BUCKET")

	// The worker→backend tool callback must be mTLS-protected in prod.
	if !c.InternalMTLSEnabled {
		missing = append(missing, "INTERNAL_MTLS_ENABLED (must be true in prod)")
	} else {
		require(c.InternalMTLSCertPath, "INTERNAL_MTLS_CERT_PATH")
		require(c.InternalMTLSKeyPath, "INTERNAL_MTLS_KEY_PATH")
		require(c.InternalMTLSClientCAPath, "INTERNAL_MTLS_CLIENT_CA_PATH")
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
