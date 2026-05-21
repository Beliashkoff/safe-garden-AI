package llm

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config — настройки фабрики LLM-клиента, читаются из env. См. CLAUDE.md
// инвариант №5 и ARCH §11.2: бэкенд не знает Anthropic-ключ, он живёт
// только в worker'е.
type Config struct {
	// Kind выбирает реализацию: "worker" (mTLS-клиент к llm-worker, prod)
	// или "mock" (фикстуры, dev/тесты).
	Kind string `envconfig:"LLM_CLIENT_KIND" default:"mock"`

	// BaseURL — адрес worker'а, например https://worker.agronomai.site:443.
	// В dev допустим http://localhost:8081 без TLS.
	BaseURL string `envconfig:"LLM_WORKER_BASE_URL" default:""`

	// MTLSEnabled включает клиентский mTLS. В prod обязателен.
	// В dev по умолчанию выключен — worker слушает чистый HTTP.
	MTLSEnabled bool `envconfig:"LLM_WORKER_MTLS_ENABLED" default:"false"`

	// Пути до клиентского сертификата + ключа и CA для проверки сервера.
	MTLSCertPath string `envconfig:"LLM_WORKER_CLIENT_CERT_PATH" default:""`
	MTLSKeyPath  string `envconfig:"LLM_WORKER_CLIENT_KEY_PATH" default:""`
	MTLSCAPath   string `envconfig:"LLM_WORKER_CA_PATH" default:""`
}

func LoadConfig() (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, fmt.Errorf("llm envconfig: %w", err)
	}
	return &c, nil
}
