//go:build integration

package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/cleanup"
)

func countRows(t *testing.T, query, userID string) int {
	t.Helper()
	var n int
	require.NoError(t, adminDB.QueryRow(query, userID).Scan(&n))
	return n
}

func TestAccountDelete_WipesContent_ThenCleanupPurgesMedia(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "wipe@example.com")
	uid := res.User.ID

	// Seed a photo message: presign → (fake) PUT → post with image_ref.
	up := h.presign(t, res.AccessToken, "image/jpeg", 1024)
	h.objs.put(up.Key, []byte("jpeg"), "image/jpeg")
	body := map[string]any{"content": []map[string]string{
		{"type": "text", "text": "diagnose"},
		{"type": "image_ref", "storage_key": up.Key},
	}}
	resp, data := h.do(t, http.MethodPost, "/v1/messages", body, bearer(res.AccessToken))
	require.Equalf(t, http.StatusOK, resp.StatusCode, "body: %s", data)

	// Sanity: content exists before deletion.
	require.Equal(t, 1, countRows(t, "SELECT count(*) FROM conversations WHERE user_id=$1::uuid", uid))
	require.Positive(t, countRows(t, "SELECT count(*) FROM messages WHERE user_id=$1::uuid", uid))
	require.Equal(t, 1, countRows(t, "SELECT count(*) FROM uploads WHERE user_id=$1::uuid", uid))

	// DELETE /account → content hard-deleted, user anonymized, media not yet purged.
	resp, _ = h.do(t, http.MethodDelete, "/v1/account", nil, bearer(res.AccessToken))
	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	assert.Zero(t, countRows(t, "SELECT count(*) FROM conversations WHERE user_id=$1::uuid", uid))
	assert.Zero(t, countRows(t, "SELECT count(*) FROM messages WHERE user_id=$1::uuid", uid))
	assert.Zero(t, countRows(t, "SELECT count(*) FROM message_blocks b JOIN messages m ON m.id=b.message_id WHERE m.user_id=$1::uuid", uid))
	assert.Zero(t, countRows(t, "SELECT count(*) FROM uploads WHERE user_id=$1::uuid", uid))

	var deletedAt, purgedAt *string
	require.NoError(t, adminDB.QueryRow(
		"SELECT deleted_at::text, media_purged_at::text FROM users WHERE id=$1::uuid", uid,
	).Scan(&deletedAt, &purgedAt))
	require.NotNil(t, deletedAt, "user soft-deleted")
	require.Nil(t, purgedAt, "media not purged yet")

	// audit row recorded.
	assert.Positive(t, countRows(t,
		"SELECT count(*) FROM audit_log WHERE user_id=$1::uuid AND action='account_deleted'", uid))

	// Run the cleanup job → prefix purged, marker set, object gone.
	svc := cleanup.NewService(testStore, h.objs, testLogger)
	require.NoError(t, svc.RunOnce(context.Background()))

	assert.True(t, h.objs.prefixDeleted("u/"+uid+"/"), "DeletePrefix called for the user")
	require.NoError(t, adminDB.QueryRow(
		"SELECT media_purged_at::text FROM users WHERE id=$1::uuid", uid,
	).Scan(&purgedAt))
	assert.NotNil(t, purgedAt, "media_purged_at set after cleanup")

	// Idempotent: a second run finds nothing pending.
	require.NoError(t, svc.RunOnce(context.Background()))
}

func TestCleanup_GCUnusedUploads(t *testing.T) {
	h := newHarness(t)
	res := h.signInEmail(t, "gc@example.com")

	// Presign creates an unused upload row; backdate it past the 7-day window.
	up := h.presign(t, res.AccessToken, "image/jpeg", 1024)
	h.objs.put(up.Key, []byte("jpeg"), "image/jpeg")
	_, err := adminDB.Exec(
		"UPDATE uploads SET created_at = NOW() - INTERVAL '8 days' WHERE storage_key=$1", up.Key)
	require.NoError(t, err)

	require.Equal(t, 1, countRowsKey(t, up.Key))

	svc := cleanup.NewService(testStore, h.objs, testLogger)
	n, err := svc.GCUnusedUploads(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, 1)

	assert.Zero(t, countRowsKey(t, up.Key), "stale unused upload row deleted")
}

func countRowsKey(t *testing.T, key string) int {
	t.Helper()
	var n int
	require.NoError(t, adminDB.QueryRow("SELECT count(*) FROM uploads WHERE storage_key=$1", key).Scan(&n))
	return n
}
