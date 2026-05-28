package chat

import "encoding/json"

// Sink receives chat stream events. The transport layer implements it over SSE
// (ARCH §4.3 event contract). Methods that write to the client return an error
// so SendMessage stops (and finalizes as cancelled) when the client disconnects;
// Failed is best-effort and never propagates.
type Sink interface {
	MessageStarted(messageID string) error
	Delta(text string) error
	ToolUse(name string, args json.RawMessage) error
	FertilizerCard(data json.RawMessage) error
	Done(messageID string, tokensIn, tokensOut int64) error
	Failed(code, msg string)
}
