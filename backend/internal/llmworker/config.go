package llmworker

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config — настройки llm-worker'а. См. ARCH §8.6 и §11.2: единственный
// сервис, в котором живёт ANTHROPIC_API_KEY и UID_HASH_PEPPER.
//
// В Этапе 0.7 (скелет, echo) поля Anthropic-аккаунта только читаются,
// но не используются — они активируются в Этапе 2.2 после подключения
// anthropic-sdk-go.
type Config struct {
	Env       string `envconfig:"ENV" default:"dev"`
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	SentryDSN string `envconfig:"SENTRY_DSN" default:""`

	HTTPHost string `envconfig:"LLM_WORKER_HTTP_HOST" default:""`
	HTTPPort int    `envconfig:"LLM_WORKER_HTTP_PORT" default:"8081"`

	// AnthropicAPIKey — единственное место в репозитории, где хранится ключ.
	// Bерхний бэкенд (cmd/api) его не знает (CLAUDE.md инвариант №5).
	// В Этапе 0.7 пустой допустим — worker работает в echo-режиме.
	AnthropicAPIKey string `envconfig:"ANTHROPIC_API_KEY" default:""`

	// UIDHashPepper — соль для sha256(uid+pepper) на стороне бэкенда.
	// В worker'е сюда попадает только для согласования формата
	// (метрики антифрода — Этап 2.3). На 0.7 — placeholder.
	UIDHashPepper string `envconfig:"UID_HASH_PEPPER" default:""`

	// BackendCallbackURL — куда worker делает tool-callback (Этап 5.2).
	BackendCallbackURL string `envconfig:"BACKEND_CALLBACK_URL" default:""`
}

func LoadConfig() (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, fmt.Errorf("llmworker envconfig: %w", err)
	}
	return &c, nil
}
