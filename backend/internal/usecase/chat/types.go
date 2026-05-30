// Package chat is the chat usecase: it persists messages, loads conversation
// history, drives the LLM client, and relays streamed events to a Sink. The
// transport layer adapts the Sink to SSE. No transport types leak here.
package chat

import "time"

// InputBlock is one content block from the client. Stage 3.1 accepts "text" and
// "image_ref" (with StorageKey); "audio_ref" and others are rejected (Stage 4+).
type InputBlock struct {
	Type       string
	Text       string
	StorageKey string
}

// SendInput is the decoded POST /v1/messages body plus request context.
type SendInput struct {
	Blocks    []InputBlock
	RequestID string
}

// BlockView is a stored content block projected for reads (no pgtype). For
// image blocks Text is empty and StorageKey points at the object.
type BlockView struct {
	Type       string
	Text       string
	StorageKey string
}

// MessageView is a stored message projected for reads.
type MessageView struct {
	ID        string
	Role      string
	Status    string
	CreatedAt time.Time
	Content   []BlockView
}

// ConversationView is the GET /v1/conversation payload.
type ConversationView struct {
	ConversationID string
	Messages       []MessageView // chronological (oldest→newest) within the page
	NextCursor     string        // "" when there is no older page
}

// MessagesPage is a keyset page of history (GET /v1/conversation/messages).
type MessagesPage struct {
	Messages   []MessageView
	NextCursor string
}
