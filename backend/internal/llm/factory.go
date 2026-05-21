package llm

import "fmt"

// New собирает реализацию Client по конфигу. Используется на старте бэкенда.
func New(cfg *Config) (Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("llm: nil config")
	}
	switch cfg.Kind {
	case "mock", "":
		return NewMockClient(), nil
	case "worker":
		return NewWorkerClient(cfg)
	default:
		return nil, fmt.Errorf("llm: unknown LLM_CLIENT_KIND %q (want \"worker\" or \"mock\")", cfg.Kind)
	}
}
