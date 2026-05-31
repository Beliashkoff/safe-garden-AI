package audio

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config selects and configures the transcription provider. Secrets come from
// env only (CLAUDE.md). Defaults keep the app booting in dev without any STT
// credentials: Kind defaults to "mock".
type Config struct {
	// Kind selects the implementation: "mock" (dev/tests), "speechkit" (Yandex
	// SpeechKit v3, prod), or "gigachat" (reserved fallback, not implemented).
	Kind string `envconfig:"STT_PROVIDER_KIND" default:"mock"`

	// SpeechKit (Yandex) — gRPC streaming recognition. The API key belongs to a
	// service account, which implies the folder, so SpeechKitFolderID is usually
	// empty (set it only for IAM-token auth).
	SpeechKitAPIKey   string `envconfig:"SPEECHKIT_API_KEY" default:""`
	SpeechKitEndpoint string `envconfig:"SPEECHKIT_ENDPOINT" default:"stt.api.cloud.yandex.net:443"`
	SpeechKitFolderID string `envconfig:"SPEECHKIT_FOLDER_ID" default:""`
	SpeechKitModel    string `envconfig:"SPEECHKIT_MODEL" default:"general"`

	// Language is passed to the recognizer as a single-entry whitelist when set.
	Language string `envconfig:"STT_LANGUAGE" default:"ru-RU"`

	// ffmpeg/ffprobe binaries for m4a->OggOpus conversion. Installed in the API
	// Docker image.
	FFmpegPath  string `envconfig:"FFMPEG_PATH" default:"ffmpeg"`
	FFprobePath string `envconfig:"FFPROBE_PATH" default:"ffprobe"`

	// GigaChat (Sber SaluteSpeech) — reserved fallback. Read from env so the
	// provider can be wired in later without code changes; unused today.
	GigaChatAPIKey string `envconfig:"GIGACHAT_API_KEY" default:""`
	GigaChatScope  string `envconfig:"GIGACHAT_SCOPE" default:""`
}

func LoadConfig() (*Config, error) {
	var c Config
	if err := envconfig.Process("", &c); err != nil {
		return nil, fmt.Errorf("audio envconfig: %w", err)
	}
	if c.Kind == "speechkit" && c.SpeechKitAPIKey == "" {
		return nil, fmt.Errorf("audio: STT_PROVIDER_KIND=speechkit requires SPEECHKIT_API_KEY")
	}
	return &c, nil
}
