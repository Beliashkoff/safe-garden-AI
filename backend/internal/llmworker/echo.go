package llmworker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// echoProvider mirrors the last user message back as word-by-word deltas. It is
// the dev fallback used when ANTHROPIC_API_KEY is unset, so the worker runs
// locally (make worker-dev) without a real key. Never used in prod (config
// validation requires the key when ENV=prod).
type echoProvider struct{}

func newEchoProvider() *echoProvider { return &echoProvider{} }

func (echoProvider) stream(ctx context.Context, req messageRequest, sink eventSink) error {
	if err := sink.started("msg_echo_" + randomID()); err != nil {
		return err
	}

	text := req.lastUserText()
	if text == "" {
		text = "(empty)"
	}
	for _, word := range splitWords(text) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := sink.delta(word); err != nil {
			return err
		}
	}

	if err := sink.usage(0, 0); err != nil {
		return err
	}
	return sink.done()
}

// splitWords keeps a leading space on every token except the first, so the
// client can concatenate deltas without extra logic.
func splitWords(s string) []string {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	for i, f := range fields {
		if i == 0 {
			out = append(out, f)
		} else {
			out = append(out, " "+f)
		}
	}
	return out
}

func randomID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}
