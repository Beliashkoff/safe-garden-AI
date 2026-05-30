//go:build integration

package handler_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sseEvent struct{ event, data string }

func readSSE(t *testing.T, r io.Reader) []sseEvent {
	t.Helper()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var (
		events []sseEvent
		cur    sseEvent
		has    bool
	)
	flush := func() {
		if cur.event != "" || has {
			events = append(events, cur)
		}
		cur, has = sseEvent{}, false
	}
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "event:"):
			cur.event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			cur.data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			has = true
		}
	}
	flush()
	return events
}

func textMessageBody(text string) map[string]any {
	return map[string]any{"content": []map[string]string{{"type": "text", "text": text}}}
}

func eventTypes(events []sseEvent) []string {
	out := make([]string, len(events))
	for i, e := range events {
		out[i] = e.event
	}
	return out
}

func TestChat_PostMessage_HappyPath(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "chat@example.com")

	resp, data := h.do(t, http.MethodPost, "/v1/messages", textMessageBody("hello"), bearer(res.AccessToken))
	require.Equalf(t, http.StatusOK, resp.StatusCode, "body: %s", data)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")

	events := readSSE(t, strings.NewReader(string(data)))
	types := eventTypes(events)
	assert.Equal(t, "message_started", types[0])
	assert.Equal(t, "done", types[len(types)-1])
	assert.Contains(t, types, "delta")

	// done carries the assistant message id + token usage.
	last := events[len(events)-1]
	assert.Contains(t, last.data, "message_id")
	assert.Contains(t, last.data, "tokens_used")

	// DB: user message + completed assistant message + usage_log row.
	var assistantStatus string
	require.NoError(t, adminDB.QueryRow(
		"SELECT status FROM messages WHERE user_id=$1::uuid AND role='assistant'", res.User.ID,
	).Scan(&assistantStatus))
	assert.Equal(t, "complete", assistantStatus)

	var assistantText string
	require.NoError(t, adminDB.QueryRow(
		`SELECT b.content_text FROM message_blocks b
		 JOIN messages m ON m.id=b.message_id
		 WHERE m.user_id=$1::uuid AND m.role='assistant'`, res.User.ID,
	).Scan(&assistantText))
	assert.Equal(t, "Mock response from llm.MockClient", assistantText)

	var usageCount int
	require.NoError(t, adminDB.QueryRow(
		"SELECT count(*) FROM usage_log WHERE user_id=$1::uuid AND endpoint='/v1/messages'", res.User.ID,
	).Scan(&usageCount))
	assert.Equal(t, 1, usageCount)
}

func TestChat_PostMessage_RejectsAudioRef(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "audio@example.com")

	// audio_ref is Stage 4 → still unsupported.
	body := map[string]any{"content": []map[string]string{{"type": "audio_ref", "storage_key": "u/x/a.m4a"}}}
	resp, data := h.do(t, http.MethodPost, "/v1/messages", body, bearer(res.AccessToken))
	require.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	assert.Contains(t, string(data), "unsupported_media_type")
}

type denyLimiter struct{}

func (denyLimiter) AllowMessage(context.Context, uuid.UUID) (bool, error) { return false, nil }

func TestChat_PostMessage_RateLimited(t *testing.T) {
	h := newHarness(t, withMessageLimiter(denyLimiter{}))
	res := h.signInEmail(t, "rl@example.com")

	resp, data := h.do(t, http.MethodPost, "/v1/messages", textMessageBody("hi"), bearer(res.AccessToken))
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	assert.Contains(t, string(data), "rate_limited")
}

func TestChat_PostMessage_CancelMarksCancelled(t *testing.T) {
	h := newHarness(t)
	h.mock.DelayPerEvent = 80 * time.Millisecond // slow stream so we can cancel mid-flight
	res := h.signInEmail(t, "cancel@example.com")

	ctx, cancel := context.WithCancel(context.Background())
	var buf strings.Builder
	_ = json.NewEncoder(&buf).Encode(textMessageBody("take your time"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.srv.URL+"/v1/messages", strings.NewReader(buf.String()))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+res.AccessToken)

	resp, err := h.srv.Client().Do(req)
	require.NoError(t, err)
	go func() {
		time.Sleep(120 * time.Millisecond) // let message_started + ~1 delta through
		cancel()
	}()
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	// Finalize runs on a detached context after the client goes away.
	require.Eventually(t, func() bool {
		var status string
		if err := adminDB.QueryRow(
			"SELECT status FROM messages WHERE user_id=$1::uuid AND role='assistant'", res.User.ID,
		).Scan(&status); err != nil {
			return false
		}
		return status == "cancelled"
	}, 3*time.Second, 50*time.Millisecond, "assistant message should be marked cancelled")
}

func TestChat_GetConversation_ReturnsHistory(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "hist@example.com")

	resp, _ := h.do(t, http.MethodPost, "/v1/messages", textMessageBody("first"), bearer(res.AccessToken))
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, data := h.do(t, http.MethodGet, "/v1/conversation", nil, bearer(res.AccessToken))
	require.Equalf(t, http.StatusOK, resp.StatusCode, "body: %s", data)
	var conv struct {
		ID       string `json:"id"`
		Messages []struct {
			ID     string `json:"id"`
			Role   string `json:"role"`
			Status string `json:"status"`
		} `json:"messages"`
	}
	require.NoError(t, json.Unmarshal(data, &conv))
	assert.NotEmpty(t, conv.ID)
	require.Len(t, conv.Messages, 2) // user + assistant, chronological
	assert.Equal(t, "user", conv.Messages[0].Role)
	assert.Equal(t, "assistant", conv.Messages[1].Role)
}

func TestChat_DeleteMessage_Ownership(t *testing.T) {
	h := newHarness(t)
	owner := h.signInEmail(t, "owner@example.com")
	resp, _ := h.do(t, http.MethodPost, "/v1/messages", textMessageBody("mine"), bearer(owner.AccessToken))
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Grab the user message id.
	_, convData := h.do(t, http.MethodGet, "/v1/conversation", nil, bearer(owner.AccessToken))
	var conv struct {
		Messages []struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"messages"`
	}
	require.NoError(t, json.Unmarshal(convData, &conv))
	var userMsgID string
	for _, m := range conv.Messages {
		if m.Role == "user" {
			userMsgID = m.ID
		}
	}
	require.NotEmpty(t, userMsgID)

	// A different user cannot delete it.
	other := h.signInEmail(t, "other@example.com")
	resp, _ = h.do(t, http.MethodDelete, "/v1/messages/"+userMsgID, nil, bearer(other.AccessToken))
	require.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Owner can.
	resp, _ = h.do(t, http.MethodDelete, "/v1/messages/"+userMsgID, nil, bearer(owner.AccessToken))
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Second delete → 404.
	resp, _ = h.do(t, http.MethodDelete, "/v1/messages/"+userMsgID, nil, bearer(owner.AccessToken))
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestChat_Endpoints_RequireAuth(t *testing.T) {
	h := newHarness(t)
	resp, _ := h.do(t, http.MethodGet, "/v1/conversation", nil, nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
