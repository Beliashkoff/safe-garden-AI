package llmworker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
)

type fakeProvider struct {
	fn func(ctx context.Context, req messageRequest, sink eventSink) error
}

func (f fakeProvider) stream(ctx context.Context, req messageRequest, sink eventSink) error {
	return f.fn(ctx, req, sink)
}

func serverWith(p provider) *httptest.Server {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := &Server{logger: logger, cfg: &Config{Env: "dev"}, provider: p}
	return httptest.NewServer(srv.Routes())
}

const validBody = `{
	"model":"claude-opus-4-7",
	"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}],
	"metadata":{"uid_hash":"abc","request_id":"req_1"}
}`

func TestHandler_StreamsProviderEvents(t *testing.T) {
	t.Parallel()
	p := fakeProvider{fn: func(_ context.Context, _ messageRequest, sink eventSink) error {
		require.NoError(t, sink.started("msg_1"))
		require.NoError(t, sink.delta("hello"))
		require.NoError(t, sink.toolUse("recommend_fertilizer", json.RawMessage(`{"problem":"leaf_yellowing"}`)))
		require.NoError(t, sink.usage(10, 20))
		return sink.done()
	}}
	ts := serverWith(p)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/v1/llm/messages", "application/json", strings.NewReader(validBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	events := parseSSEStream(t, resp.Body)
	types := make([]string, len(events))
	for i, e := range events {
		types[i] = e.event
	}
	assert.Equal(t, []string{
		string(llm.EventMessageStarted),
		string(llm.EventDelta),
		string(llm.EventToolUse),
		string(llm.EventUsage),
		string(llm.EventDone),
	}, types)
	assert.Contains(t, events[2].data, "recommend_fertilizer")
	assert.Contains(t, events[3].data, "10")
	assert.Contains(t, events[3].data, "20")
}

func TestHandler_ProviderFailureEmitsErrorEvent(t *testing.T) {
	t.Parallel()
	p := fakeProvider{fn: func(_ context.Context, _ messageRequest, sink eventSink) error {
		require.NoError(t, sink.started("msg_1"))
		sink.failed("upstream_error", "model unavailable")
		return errors.New("boom")
	}}
	ts := serverWith(p)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/v1/llm/messages", "application/json", strings.NewReader(validBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	events := parseSSEStream(t, resp.Body)
	var sawError bool
	for _, e := range events {
		if e.event == string(llm.EventError) {
			sawError = true
			assert.Contains(t, e.data, "upstream_error")
		}
	}
	assert.True(t, sawError, "error event must be emitted on provider failure")
}

func TestHandler_InvalidPayloadSkipsProvider(t *testing.T) {
	t.Parallel()
	called := false
	p := fakeProvider{fn: func(context.Context, messageRequest, eventSink) error {
		called = true
		return nil
	}}
	ts := serverWith(p)
	defer ts.Close()

	// Banned top-level field → 400 before the provider runs.
	resp, err := http.Post(ts.URL+"/v1/llm/messages", "application/json",
		strings.NewReader(`{"messages":[{"role":"user","content":[{"type":"text","text":"x"}]}],"email":"a@b.c"}`))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.False(t, called, "provider must not run on invalid payload")
}
