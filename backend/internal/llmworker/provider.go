package llmworker

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
)

// eventSink abstracts the SSE output (ARCH §11.3 event contract) so providers
// emit events without knowing about http.ResponseWriter. Methods that write to
// the client return an error (so a provider stops when the client disconnects);
// failed is best-effort and never propagates.
type eventSink interface {
	started(messageID string) error
	delta(text string) error
	toolUse(name string, args json.RawMessage) error
	usage(tokensIn, tokensOut int64) error
	done() error
	failed(code, msg string)
}

// provider runs a single completion, emitting events through sink until done or
// error. It returns an error for transport/client failures (e.g. the client
// went away mid-stream); upstream model failures are emitted via sink.failed.
//
// Two implementations: anthropicProvider (real Claude via anthropic-sdk-go) and
// echoProvider (dev fallback when no API key). A future openrouter provider
// (ARCH §11.5) slots in here unchanged.
type provider interface {
	stream(ctx context.Context, req messageRequest, sink eventSink) error
}

// sseSink writes events to the HTTP response as Server-Sent Events.
type sseSink struct {
	w http.ResponseWriter
}

func (s *sseSink) started(messageID string) error {
	return writeSSE(s.w, string(llm.EventMessageStarted), map[string]string{"message_id": messageID})
}

func (s *sseSink) delta(text string) error {
	return writeSSE(s.w, string(llm.EventDelta), map[string]string{"text": text})
}

func (s *sseSink) toolUse(name string, args json.RawMessage) error {
	return writeSSE(s.w, string(llm.EventToolUse), map[string]any{"tool": name, "args": args})
}

func (s *sseSink) usage(tokensIn, tokensOut int64) error {
	return writeSSE(s.w, string(llm.EventUsage), map[string]int64{"tokens_in": tokensIn, "tokens_out": tokensOut})
}

func (s *sseSink) done() error {
	return writeSSE(s.w, string(llm.EventDone), struct{}{})
}

func (s *sseSink) failed(code, msg string) {
	_ = writeSSE(s.w, string(llm.EventError), map[string]string{"code": code, "message": msg})
}
