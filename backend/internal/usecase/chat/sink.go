package chat

import "encoding/json"

// Sink receives chat stream events. The transport layer implements it over SSE
// (ARCH §4.3 event contract). Methods that write to the client return an error
// so SendMessage stops (and finalizes as cancelled) when the client disconnects;
// Failed is best-effort and never propagates.
type Sink interface {
	MessageStarted(messageID string) error
	// Transcription reports the server-side transcript of a user voice message,
	// emitted before the assistant reply starts so the client can render the
	// recognized text under the player.
	Transcription(messageID, storageKey, text string, durationMs int64) error
	Delta(text string) error
	ToolUse(name string, args json.RawMessage) error
	FertilizerCard(data json.RawMessage) error
	Done(messageID string, tokensIn, tokensOut int64) error
	Failed(code, msg string)
}
