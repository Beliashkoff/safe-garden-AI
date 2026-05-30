//go:build integration

package handler_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type presignResp struct {
	URL       string            `json:"url"`
	Key       string            `json:"key"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

func (h *harness) presign(t *testing.T, token, contentType string, size int) presignResp {
	t.Helper()
	body := map[string]any{"content_type": contentType, "size_bytes": size}
	resp, data := h.do(t, http.MethodPost, "/v1/uploads/presign", body, bearer(token))
	require.Equalf(t, http.StatusOK, resp.StatusCode, "body: %s", data)
	var out presignResp
	require.NoError(t, json.Unmarshal(data, &out))
	return out
}

func TestUploads_Presign_HappyPath(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "presign@example.com")

	out := h.presign(t, res.AccessToken, "image/jpeg", 1024)

	assert.True(t, strings.HasPrefix(out.Key, "u/"+res.User.ID+"/img/"), "key: %s", out.Key)
	assert.True(t, strings.HasSuffix(out.Key, ".jpg"))
	assert.NotEmpty(t, out.URL)
	assert.Equal(t, "image/jpeg", out.Headers["Content-Type"])
	assert.True(t, out.ExpiresAt.After(time.Now()))

	var used bool
	require.NoError(t, adminDB.QueryRow(
		"SELECT used FROM uploads WHERE storage_key=$1", out.Key,
	).Scan(&used))
	assert.False(t, used, "freshly presigned upload is unused")
}

func TestUploads_Presign_RejectsBadType(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "badtype@example.com")

	resp, data := h.do(t, http.MethodPost, "/v1/uploads/presign",
		map[string]any{"content_type": "application/pdf", "size_bytes": 10}, bearer(res.AccessToken))
	require.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	assert.Contains(t, string(data), "unsupported")
}

func TestUploads_Presign_RejectsTooLarge(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "toolarge@example.com")

	resp, _ := h.do(t, http.MethodPost, "/v1/uploads/presign",
		map[string]any{"content_type": "image/jpeg", "size_bytes": 11 * 1024 * 1024}, bearer(res.AccessToken))
	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
}

func TestUploads_Presign_RequiresAuth(t *testing.T) {
	h := newHarness(t)
	resp, _ := h.do(t, http.MethodPost, "/v1/uploads/presign",
		map[string]any{"content_type": "image/jpeg", "size_bytes": 10}, nil)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestChat_PostMessage_WithImage(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "photo@example.com")

	// 1) presign, 2) the client would PUT to storage — here we seed the fake.
	up := h.presign(t, res.AccessToken, "image/jpeg", 1024)
	h.objs.put(up.Key, []byte("fake-jpeg-bytes"), "image/jpeg")

	// 3) send a message referencing the image.
	body := map[string]any{"content": []map[string]string{
		{"type": "text", "text": "what is wrong with this plant?"},
		{"type": "image_ref", "storage_key": up.Key},
	}}
	resp, data := h.do(t, http.MethodPost, "/v1/messages", body, bearer(res.AccessToken))
	require.Equalf(t, http.StatusOK, resp.StatusCode, "body: %s", data)
	types := eventTypes(readSSE(t, strings.NewReader(string(data))))
	assert.Equal(t, "message_started", types[0])
	assert.Equal(t, "done", types[len(types)-1])

	// 4) DB: an image block was stored for the user message.
	var blockType, storageKey string
	require.NoError(t, adminDB.QueryRow(
		`SELECT b.type, b.storage_key FROM message_blocks b
		 JOIN messages m ON m.id=b.message_id
		 WHERE m.user_id=$1::uuid AND m.role='user' AND b.type='image'`, res.User.ID,
	).Scan(&blockType, &storageKey))
	assert.Equal(t, "image", blockType)
	assert.Equal(t, up.Key, storageKey)

	// 5) the upload is marked used, and hydration fetched it for the model.
	var used bool
	require.NoError(t, adminDB.QueryRow(
		"SELECT used FROM uploads WHERE storage_key=$1", up.Key,
	).Scan(&used))
	assert.True(t, used)
	assert.GreaterOrEqual(t, h.objs.getCount(up.Key), 1, "hydration should read the image")
}

func TestChat_PostMessage_RejectsForeignImage(t *testing.T) {
	h := newHarness(t)
	owner := h.signInEmail(t, "owner-img@example.com")
	up := h.presign(t, owner.AccessToken, "image/jpeg", 10)

	// A different user references the owner's key → 404 (prefix/ownership).
	other := h.signInEmail(t, "other-img@example.com")
	body := map[string]any{"content": []map[string]string{{"type": "image_ref", "storage_key": up.Key}}}
	resp, data := h.do(t, http.MethodPost, "/v1/messages", body, bearer(other.AccessToken))
	require.Equalf(t, http.StatusNotFound, resp.StatusCode, "body: %s", data)
	assert.Contains(t, string(data), "not_found")
}
