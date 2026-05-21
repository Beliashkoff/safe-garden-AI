package llm

import (
	"context"
	"encoding/json"
)

// Client — интерфейс LLM-провайдера, потребляется usecase-слоем
// (`internal/usecase/chat` в Этапе 2.3). Стриминг возвращается каналом —
// получатель читает события до закрытия канала, отмена через ctx.
type Client interface {
	Send(ctx context.Context, req SendRequest) (<-chan StreamEvent, error)
}

// Message — нейтральная (не Anthropic-специфичная) модель сообщения.
// В worker'е она будет конвертирована в anthropic.MessageParam (Этап 2.2).
type Message struct {
	Role    string         `json:"role"` // "user" | "assistant"
	Content []MessageBlock `json:"content"`
}

// MessageBlock — мультимодальный блок (см. ARCH §6.1 message_blocks).
type MessageBlock struct {
	Type      string          `json:"type"` // "text" | "image" | "audio" | "transcription" | "tool_use" | "tool_result"
	Text      string          `json:"text,omitempty"`
	MediaB64  string          `json:"media_b64,omitempty"`
	MediaType string          `json:"media_type,omitempty"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// Tool — определение Claude tool (см. ARCH §7.2).
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// Metadata — обезличенные метаданные запроса. Только uid_hash и request_id
// (CLAUDE.md инвариант №5). В worker email/uuid/токены не передаём.
type Metadata struct {
	UIDHash   string `json:"uid_hash,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// SendRequest — полезная нагрузка для Send().
type SendRequest struct {
	Model    string    `json:"model"`
	System   string    `json:"system,omitempty"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
	Metadata Metadata  `json:"metadata,omitempty"`
}

// StreamEvent — одно SSE-событие.
type StreamEvent struct {
	Type EventType
	Data json.RawMessage
}
