package llmworker

import (
	"encoding/json"
	"errors"
	"fmt"
)

// messageRequest — payload `POST /v1/llm/messages`. Зеркало llm.SendRequest
// в формате wire-протокола (ARCH §11.3). Используется только внутри worker'а;
// в основной код типы переэкспортируются через `internal/llm`.
type messageRequest struct {
	Model    string        `json:"model"`
	System   string        `json:"system,omitempty"`
	Messages []messageItem `json:"messages"`
	Tools    []toolDef     `json:"tools,omitempty"`
	Metadata requestMeta   `json:"metadata,omitempty"`
}

type messageItem struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	MediaB64  string          `json:"media_b64,omitempty"`
	MediaType string          `json:"media_type,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type toolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// requestMeta — обезличенные метаданные. Allowlist: uid_hash, request_id.
// Любые другие поля → ошибка валидации (защита инварианта №5).
type requestMeta struct {
	UIDHash   string `json:"uid_hash,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// validate проверяет, что в payload не пришло то, что в worker'е появляться
// не должно: реальный email, user_id, refresh_token. CLAUDE.md инвариант №5.
func (r *messageRequest) validate(rawBody []byte) error {
	// Декодируем «сырой» payload во второй раз, чтобы поймать
	// неизвестные top-level и metadata-поля.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	for _, banned := range []string{"email", "user_id", "refresh_token", "access_token"} {
		if _, ok := raw[banned]; ok {
			return fmt.Errorf("field %q must not be sent to worker (CLAUDE.md invariant #5)", banned)
		}
	}

	if metaRaw, ok := raw["metadata"]; ok && len(metaRaw) > 0 {
		var meta map[string]json.RawMessage
		if err := json.Unmarshal(metaRaw, &meta); err != nil {
			return fmt.Errorf("invalid metadata: %w", err)
		}
		allowed := map[string]struct{}{"uid_hash": {}, "request_id": {}}
		for key := range meta {
			if _, ok := allowed[key]; !ok {
				return fmt.Errorf("metadata field %q not allowed (allowlist: uid_hash, request_id)", key)
			}
		}
	}

	if len(r.Messages) == 0 {
		return errors.New("messages must not be empty")
	}
	return nil
}

// lastUserText вытаскивает текст из последнего user-message — для echo.
func (r *messageRequest) lastUserText() string {
	for i := len(r.Messages) - 1; i >= 0; i-- {
		if r.Messages[i].Role != "user" {
			continue
		}
		for _, b := range r.Messages[i].Content {
			if b.Type == "text" && b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}
