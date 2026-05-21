package llmworker

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := New(&Config{Env: "dev"}, logger)
	return httptest.NewServer(srv.Routes())
}

func TestHealthz(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestHandleMessages_EchoSSE(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	defer srv.Close()

	body := strings.NewReader(`{
		"model":"echo",
		"messages":[{"role":"user","content":[{"type":"text","text":"hello world"}]}],
		"metadata":{"uid_hash":"abc","request_id":"req_1"}
	}`)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/llm/messages", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	events := parseSSEStream(t, resp.Body)
	gotTypes := make([]string, 0, len(events))
	gotData := make([]string, 0, len(events))
	for _, e := range events {
		gotTypes = append(gotTypes, e.event)
		gotData = append(gotData, e.data)
	}
	assert.Equal(t, []string{
		string(llm.EventMessageStarted),
		string(llm.EventDelta),
		string(llm.EventDelta),
		string(llm.EventUsage),
		string(llm.EventDone),
	}, gotTypes)

	assert.Contains(t, gotData[1], "hello")
	assert.Contains(t, gotData[2], "world")
}

func TestHandleMessages_RejectsBannedTopLevelField(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	defer srv.Close()

	body := strings.NewReader(`{
		"messages":[{"role":"user","content":[{"type":"text","text":"x"}]}],
		"email":"a@b.com"
	}`)
	resp, err := http.Post(srv.URL+"/v1/llm/messages", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandleMessages_RejectsBannedMetadataField(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	defer srv.Close()

	body := strings.NewReader(`{
		"messages":[{"role":"user","content":[{"type":"text","text":"x"}]}],
		"metadata":{"email":"a@b.com"}
	}`)
	resp, err := http.Post(srv.URL+"/v1/llm/messages", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandleMessages_EmptyMessagesRejected(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	defer srv.Close()

	body := strings.NewReader(`{"messages":[]}`)
	resp, err := http.Post(srv.URL+"/v1/llm/messages", "application/json", body)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestHandleMessages_CancelStopsStream(t *testing.T) {
	t.Parallel()
	srv := newTestServer(t)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	body := strings.NewReader(`{
		"messages":[{"role":"user","content":[{"type":"text","text":"one two three four"}]}]
	}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/v1/llm/messages", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	// Канал должен закрыться без panic; нам неважно, сколько событий пришло.
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

type sseEvent struct{ event, data string }

func parseSSEStream(t *testing.T, r io.Reader) []sseEvent {
	t.Helper()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var (
		events  []sseEvent
		cur     sseEvent
		hasData bool
	)
	flush := func() {
		if cur.event != "" || hasData {
			events = append(events, cur)
		}
		cur = sseEvent{}
		hasData = false
	}
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "event:"):
			cur.event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			cur.data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			hasData = true
		}
	}
	flush()
	return events
}
