//go:build integration

package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assistantMessageID sends one message (which creates a user + assistant pair)
// and returns the assistant message id, so feedback tests have a real target.
func assistantMessageID(t *testing.T, h *harness, token string) string {
	t.Helper()
	resp, _ := h.do(t, http.MethodPost, "/v1/messages", textMessageBody("how is my plant"), bearer(token))
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_, convData := h.do(t, http.MethodGet, "/v1/conversation", nil, bearer(token))
	var conv struct {
		Messages []struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"messages"`
	}
	require.NoError(t, json.Unmarshal(convData, &conv))
	for _, m := range conv.Messages {
		if m.Role == "assistant" {
			return m.ID
		}
	}
	t.Fatal("no assistant message in conversation")
	return ""
}

func feedbackBody(value any) map[string]any {
	return map[string]any{"value": value}
}

func TestChat_Feedback_UpDownAndClear(t *testing.T) {
	h := newHarness(t)
	user := h.signInEmail(t, "fb-vote@example.com")
	msgID := assistantMessageID(t, h, user.AccessToken)
	path := "/v1/messages/" + msgID + "/feedback"

	// Up.
	resp, data := h.do(t, http.MethodPut, path, feedbackBody("up"), bearer(user.AccessToken))
	require.Equalf(t, http.StatusNoContent, resp.StatusCode, "body: %s", data)
	assert.Equal(t, "up", feedbackValue(t, msgID, user.User.ID))

	// Idempotent re-up.
	resp, _ = h.do(t, http.MethodPut, path, feedbackBody("up"), bearer(user.AccessToken))
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "up", feedbackValue(t, msgID, user.User.ID))

	// Switch to down (upsert).
	resp, _ = h.do(t, http.MethodPut, path, feedbackBody("down"), bearer(user.AccessToken))
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "down", feedbackValue(t, msgID, user.User.ID))

	// Clear with explicit null.
	resp, _ = h.do(t, http.MethodPut, path, feedbackBody(nil), bearer(user.AccessToken))
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "", feedbackValue(t, msgID, user.User.ID), "verdict row must be gone after clear")

	// Clearing again is idempotent.
	resp, _ = h.do(t, http.MethodPut, path, feedbackBody(nil), bearer(user.AccessToken))
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestChat_Feedback_InvalidValue(t *testing.T) {
	h := newHarness(t)
	user := h.signInEmail(t, "fb-bad-value@example.com")
	msgID := assistantMessageID(t, h, user.AccessToken)

	resp, data := h.do(t, http.MethodPut, "/v1/messages/"+msgID+"/feedback",
		feedbackBody("meh"), bearer(user.AccessToken))
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(data), "validation_failed")
}

func TestChat_Feedback_BadID(t *testing.T) {
	h := newHarness(t)
	user := h.signInEmail(t, "fb-bad-id@example.com")

	resp, data := h.do(t, http.MethodPut, "/v1/messages/not-a-uuid/feedback",
		feedbackBody("up"), bearer(user.AccessToken))
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Contains(t, string(data), "validation_failed")
}

func TestChat_Feedback_ForeignMessageIsNotFound(t *testing.T) {
	h := newHarness(t)
	owner := h.signInEmail(t, "fb-owner@example.com")
	msgID := assistantMessageID(t, h, owner.AccessToken)

	other := h.signInEmail(t, "fb-other@example.com")
	resp, data := h.do(t, http.MethodPut, "/v1/messages/"+msgID+"/feedback",
		feedbackBody("up"), bearer(other.AccessToken))
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, string(data), "not_found")
}

func TestChat_Feedback_UserMessageIsNotFound(t *testing.T) {
	h := newHarness(t)
	user := h.signInEmail(t, "fb-usermsg@example.com")

	resp, _ := h.do(t, http.MethodPost, "/v1/messages", textMessageBody("mine"), bearer(user.AccessToken))
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_, convData := h.do(t, http.MethodGet, "/v1/conversation", nil, bearer(user.AccessToken))
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

	// Feedback only applies to assistant messages — a user message is treated as
	// not found so the verdict target stays unambiguous.
	resp, data := h.do(t, http.MethodPut, "/v1/messages/"+userMsgID+"/feedback",
		feedbackBody("up"), bearer(user.AccessToken))
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Contains(t, string(data), "not_found")
}

func TestChat_Feedback_RequiresAuth(t *testing.T) {
	h := newHarness(t)
	resp, _ := h.do(t, http.MethodPut,
		"/v1/messages/00000000-0000-0000-0000-000000000000/feedback",
		feedbackBody("up"), nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// feedbackValue returns the stored verdict for (message, user), or "" if none.
func feedbackValue(t *testing.T, messageID, userID string) string {
	t.Helper()
	var value string
	err := adminDB.QueryRow(
		"SELECT value FROM message_feedback WHERE message_id=$1::uuid AND user_id=$2::uuid",
		messageID, userID,
	).Scan(&value)
	if err != nil {
		return ""
	}
	return value
}
