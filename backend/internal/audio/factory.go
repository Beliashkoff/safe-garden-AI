package audio

import "fmt"

// New builds a Transcriber from config. Used at backend startup. Mirrors llm.New.
func New(cfg *Config) (Transcriber, error) {
	if cfg == nil {
		return nil, fmt.Errorf("audio: nil config")
	}
	switch cfg.Kind {
	case "mock", "":
		return NewMockTranscriber(), nil
	case "speechkit":
		return NewSpeechKitTranscriber(cfg)
	case "gigachat":
		return NewGigaChatTranscriber(cfg), nil
	default:
		return nil, fmt.Errorf("audio: unknown STT_PROVIDER_KIND %q (want \"speechkit\", \"gigachat\" or \"mock\")", cfg.Kind)
	}
}
