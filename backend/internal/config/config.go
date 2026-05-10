package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Env       string `envconfig:"ENV" default:"dev"`
	HTTPHost  string `envconfig:"HTTP_HOST" default:""`
	HTTPPort  int    `envconfig:"HTTP_PORT" default:"8080"`
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	SentryDSN string `envconfig:"SENTRY_DSN" default:""`
}

func Load() (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, fmt.Errorf("envconfig: %w", err)
	}
	return &c, nil
}
