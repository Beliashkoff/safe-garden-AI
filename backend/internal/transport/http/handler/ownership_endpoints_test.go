//go:build integration

package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Resource ownership (ROADMAP §6.1) is already exercised by:
//   - TestUploads_View_RejectsForeignKey       — foreign storage_key  → 403
//   - TestChat_PostMessage_RejectsForeignImage — foreign image key    → 404
//   - TestChat_PostMessage_RejectsForeignAudio — foreign audio key    → 404
//   - TestChat_DeleteMessage_Ownership         — foreign message id   → 404
//
// This file closes the remaining gap: conversation isolation. There is no
// conversation_id in the API (CLAUDE.md invariant #1: one chat per user), so the
// guard is that each user's GET /conversation returns only their own chat. This
// test fails if a future change lets the conversation lookup ignore user_id.
func TestChat_Conversation_IsolatedBetweenUsers(t *testing.T) {
	h := newHarness(t)

	alice := h.signInEmail(t, "iso-alice@example.com")
	bob := h.signInEmail(t, "iso-bob@example.com")

	respA, dataA := h.do(t, http.MethodPost, "/v1/messages", textMessageBody("alice secret"), bearer(alice.AccessToken))
	require.Equalf(t, http.StatusOK, respA.StatusCode, "body: %s", dataA)
	respB, dataB := h.do(t, http.MethodPost, "/v1/messages", textMessageBody("bob secret"), bearer(bob.AccessToken))
	require.Equalf(t, http.StatusOK, respB.StatusCode, "body: %s", dataB)

	type convResp struct {
		ID       string `json:"id"`
		Messages []struct {
			Role string `json:"role"`
		} `json:"messages"`
	}
	get := func(token string) convResp {
		_, data := h.do(t, http.MethodGet, "/v1/conversation", nil, bearer(token))
		var c convResp
		require.NoError(t, json.Unmarshal(data, &c))
		return c
	}

	ca := get(alice.AccessToken)
	cb := get(bob.AccessToken)

	// Distinct conversations (unique index conversations(user_id)).
	assert.NotEqual(t, ca.ID, cb.ID, "each user must own a separate conversation")
	// Each sees only their own message pair (user + assistant) — never both.
	assert.Len(t, ca.Messages, 2, "alice must see only her own history")
	assert.Len(t, cb.Messages, 2, "bob must see only his own history")
}
