//go:build integration

package storage

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// ----- helpers --------------------------------------------------------------

func createUser(t *testing.T) uuid.UUID {
	t.Helper()
	u, err := testStore.CreateUser(testCtx, db.CreateUserParams{
		Email:         textPtr(fmt.Sprintf("chat-%s@example.com", uuid.NewString())),
		EmailVerified: true,
		Column6:       "",
	})
	require.NoError(t, err)
	return u.ID
}

func i4(n int32) pgtype.Int4 { return pgtype.Int4{Int32: n, Valid: true} }

func numeric(t *testing.T, s string) pgtype.Numeric {
	t.Helper()
	var n pgtype.Numeric
	require.NoError(t, n.Scan(s))
	return n
}

func tableCount(t *testing.T, sql string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, testStore.pool.QueryRow(testCtx, sql, args...).Scan(&n))
	return n
}

// ----- conversations --------------------------------------------------------

func TestGetOrCreateConversation_Idempotent(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user := createUser(t)

	c1, err := testStore.GetOrCreateConversation(testCtx, user)
	require.NoError(t, err)
	c2, err := testStore.GetOrCreateConversation(testCtx, user)
	require.NoError(t, err)

	assert.Equal(t, c1.ID, c2.ID, "one chat per user")

	got, err := testStore.GetConversationByUser(testCtx, user)
	require.NoError(t, err)
	assert.Equal(t, c1.ID, got.ID)
}

func TestConversation_UniquePerUser(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user := createUser(t)

	_, err := testStore.GetOrCreateConversation(testCtx, user)
	require.NoError(t, err)

	// A raw second insert must violate the unique index.
	_, err = testStore.pool.Exec(testCtx,
		"INSERT INTO conversations (user_id) VALUES ($1)", user)
	require.Error(t, err)
}

// ----- messages -------------------------------------------------------------

func newConversation(t *testing.T) (user, conv uuid.UUID) {
	t.Helper()
	user = createUser(t)
	c, err := testStore.GetOrCreateConversation(testCtx, user)
	require.NoError(t, err)
	return user, c.ID
}

func TestMessages_CreateAndComplete(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user, conv := newConversation(t)

	um, err := testStore.CreateMessage(testCtx, db.CreateMessageParams{
		ConversationID: conv, UserID: user, Role: "user", Status: "complete",
	})
	require.NoError(t, err)
	assert.Equal(t, "complete", um.Status)

	am, err := testStore.CreateMessage(testCtx, db.CreateMessageParams{
		ConversationID: conv, UserID: user, Role: "assistant", Status: "pending",
	})
	require.NoError(t, err)

	require.NoError(t, testStore.CompleteMessage(testCtx, db.CompleteMessageParams{
		ID: am.ID, TokensIn: i4(120), TokensOut: i4(340),
	}))

	got, err := testStore.GetMessageByID(testCtx, am.ID)
	require.NoError(t, err)
	assert.Equal(t, "complete", got.Status)
	assert.Equal(t, int32(120), got.TokensIn.Int32)
	assert.Equal(t, int32(340), got.TokensOut.Int32)
}

func TestMessages_StatusTransition(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user, conv := newConversation(t)
	m, err := testStore.CreateMessage(testCtx, db.CreateMessageParams{
		ConversationID: conv, UserID: user, Role: "assistant", Status: "pending",
	})
	require.NoError(t, err)

	require.NoError(t, testStore.UpdateMessageStatus(testCtx, db.UpdateMessageStatusParams{
		ID: m.ID, Status: "cancelled",
	}))
	got, err := testStore.GetMessageByID(testCtx, m.ID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", got.Status)
}

func TestMessages_KeysetPaginationWithTiebreak(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user, conv := newConversation(t)

	// Three messages created in one tx share NOW() → identical created_at,
	// forcing the id tiebreaker in keyset pagination.
	var created []db.Message
	require.NoError(t, testStore.ExecTx(testCtx, func(q *db.Queries) error {
		for i := 0; i < 3; i++ {
			m, e := q.CreateMessage(testCtx, db.CreateMessageParams{
				ConversationID: conv, UserID: user, Role: "user", Status: "complete",
			})
			if e != nil {
				return e
			}
			created = append(created, m)
		}
		return nil
	}))
	// Confirm the tie actually happened.
	require.Equal(t, created[0].CreatedAt.Time, created[2].CreatedAt.Time)

	page1, err := testStore.ListRecentMessages(testCtx, db.ListRecentMessagesParams{
		ConversationID: conv, Limit: 2,
	})
	require.NoError(t, err)
	require.Len(t, page1, 2)

	cursor := page1[len(page1)-1]
	page2, err := testStore.ListMessagesBefore(testCtx, db.ListMessagesBeforeParams{
		ConversationID:  conv,
		BeforeCreatedAt: cursor.CreatedAt,
		BeforeID:        cursor.ID,
		Limit:           2,
	})
	require.NoError(t, err)
	require.Len(t, page2, 1)

	seen := map[uuid.UUID]bool{}
	for _, m := range append(page1, page2...) {
		assert.False(t, seen[m.ID], "no row appears on two pages")
		seen[m.ID] = true
	}
	assert.Len(t, seen, 3, "pagination covers every message exactly once")
}

func TestMessages_DeleteOwnershipAndBlockCascade(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user, conv := newConversation(t)
	other := createUser(t)

	m, err := testStore.CreateMessage(testCtx, db.CreateMessageParams{
		ConversationID: conv, UserID: user, Role: "user", Status: "complete",
	})
	require.NoError(t, err)
	_, err = testStore.CreateMessageBlock(testCtx, db.CreateMessageBlockParams{
		MessageID: m.ID, OrderIndex: 0, Type: "text", ContentText: textPtr("hello"),
	})
	require.NoError(t, err)

	// Not the owner → 0 rows, message untouched.
	rows, err := testStore.DeleteMessage(testCtx, db.DeleteMessageParams{ID: m.ID, UserID: other})
	require.NoError(t, err)
	assert.Equal(t, int64(0), rows)

	// Owner → 1 row, blocks cascade.
	rows, err = testStore.DeleteMessage(testCtx, db.DeleteMessageParams{ID: m.ID, UserID: user})
	require.NoError(t, err)
	assert.Equal(t, int64(1), rows)

	blocks, err := testStore.ListBlocksByMessageIDs(testCtx, []uuid.UUID{m.ID})
	require.NoError(t, err)
	assert.Empty(t, blocks)
}

// ----- message_blocks -------------------------------------------------------

func TestMessageBlocks_OrderingAndMetadata(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user, conv := newConversation(t)
	m, err := testStore.CreateMessage(testCtx, db.CreateMessageParams{
		ConversationID: conv, UserID: user, Role: "assistant", Status: "complete",
	})
	require.NoError(t, err)

	// Insert out of order; expect ordered read.
	for _, idx := range []int32{2, 0, 1} {
		_, err := testStore.CreateMessageBlock(testCtx, db.CreateMessageBlockParams{
			MessageID: m.ID, OrderIndex: idx, Type: "text",
			ContentText: textPtr(fmt.Sprintf("block-%d", idx)),
		})
		require.NoError(t, err)
	}
	// One tool_use block with JSONB metadata.
	_, err = testStore.CreateMessageBlock(testCtx, db.CreateMessageBlockParams{
		MessageID: m.ID, OrderIndex: 3, Type: "tool_use",
		Metadata: []byte(`{"tool":"recommend_fertilizer","args":{"problem":"leaf_yellowing"}}`),
	})
	require.NoError(t, err)

	blocks, err := testStore.ListBlocksByMessageIDs(testCtx, []uuid.UUID{m.ID})
	require.NoError(t, err)
	require.Len(t, blocks, 4)
	for i := range blocks {
		assert.Equal(t, int32(i), blocks[i].OrderIndex, "ordered by order_index")
	}

	var meta map[string]any
	require.NoError(t, json.Unmarshal(blocks[3].Metadata, &meta))
	assert.Equal(t, "recommend_fertilizer", meta["tool"])
}

// ----- uploads --------------------------------------------------------------

func TestUploads_CRUD(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user := createUser(t)
	key := fmt.Sprintf("u/%s/img/x.jpg", user)

	up, err := testStore.CreateUpload(testCtx, db.CreateUploadParams{
		UserID: user, StorageKey: key, ContentType: "image/jpeg", SizeBytes: 12345,
	})
	require.NoError(t, err)
	assert.False(t, up.Used)

	got, err := testStore.GetUploadByStorageKey(testCtx, key)
	require.NoError(t, err)
	assert.Equal(t, user, got.UserID)

	require.NoError(t, testStore.MarkUploadUsed(testCtx, key))
	got, err = testStore.GetUploadByStorageKey(testCtx, key)
	require.NoError(t, err)
	assert.True(t, got.Used)

	// storage_key is unique.
	_, err = testStore.CreateUpload(testCtx, db.CreateUploadParams{
		UserID: user, StorageKey: key, ContentType: "image/jpeg", SizeBytes: 1,
	})
	require.Error(t, err)
}

// ----- fertilizers ----------------------------------------------------------

func seedFertilizer(t *testing.T, slug string, problems, plants []string, priority int32, active bool) {
	t.Helper()
	_, err := testStore.UpsertFertilizerBySlug(testCtx, db.UpsertFertilizerBySlugParams{
		Slug: slug, Name: slug, ShortDesc: "d", Category: "npk",
		Problems: problems, Plants: plants, Priority: i4(priority), Active: active,
	})
	require.NoError(t, err)
}

func TestFertilizers_UpsertAndRecommend(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })

	seedFertilizer(t, "iron-mix", []string{"leaf_yellowing", "iron_deficiency"}, []string{"tomato"}, 100, true)
	seedFertilizer(t, "universal-green", []string{"leaf_yellowing"}, nil, 50, true)
	seedFertilizer(t, "cucumber-only", []string{"leaf_yellowing"}, []string{"cucumber"}, 200, true)
	seedFertilizer(t, "inactive", []string{"leaf_yellowing"}, nil, 999, false)

	// Upsert updates in place (no duplicate; updated_at bumps).
	before, err := testStore.GetFertilizerBySlug(testCtx, "iron-mix")
	require.NoError(t, err)
	time.Sleep(5 * time.Millisecond)
	seedFertilizer(t, "iron-mix", []string{"leaf_yellowing", "iron_deficiency"}, []string{"tomato"}, 110, true)
	after, err := testStore.GetFertilizerBySlug(testCtx, "iron-mix")
	require.NoError(t, err)
	assert.Equal(t, before.ID, after.ID)
	assert.Equal(t, int32(110), after.Priority.Int32)
	assert.True(t, after.UpdatedAt.Time.After(before.UpdatedAt.Time))

	// plant=tomato: matches iron-mix (tomato) + universal-green (NULL plants),
	// excludes cucumber-only and inactive. Ordered by priority DESC.
	rec, err := testStore.RecommendFertilizers(testCtx, db.RecommendFertilizersParams{
		Problem: "leaf_yellowing", Plant: textPtr("tomato"),
	})
	require.NoError(t, err)
	slugs := make([]string, len(rec))
	for i, r := range rec {
		slugs[i] = r.Slug
	}
	assert.Equal(t, []string{"iron-mix", "universal-green"}, slugs)

	// plant=NULL: no crop filter → all active problem matches, priority DESC, max 3.
	recAll, err := testStore.RecommendFertilizers(testCtx, db.RecommendFertilizersParams{
		Problem: "leaf_yellowing",
	})
	require.NoError(t, err)
	require.Len(t, recAll, 3)
	assert.Equal(t, "cucumber-only", recAll[0].Slug, "highest priority first")
}

// ----- usage_log ------------------------------------------------------------

func TestUsageLog_InsertAndSum(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user := createUser(t)
	since := time.Now().Add(-time.Hour)

	require.NoError(t, testStore.InsertUsage(testCtx, db.InsertUsageParams{
		UserID: user, Endpoint: "/v1/messages",
		TokensIn: i4(100), TokensOut: i4(200), CostUsd: numeric(t, "0.001500"),
	}))
	require.NoError(t, testStore.InsertUsage(testCtx, db.InsertUsageParams{
		UserID: user, Endpoint: "/v1/messages",
		TokensIn: i4(50), TokensOut: i4(70), CostUsd: numeric(t, "0.000800"),
	}))

	sum, err := testStore.SumUserTokensSince(testCtx, db.SumUserTokensSinceParams{
		UserID: user, CreatedAt: tsAt(since),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(150), sum.TokensIn)
	assert.Equal(t, int64(270), sum.TokensOut)

	// cost_usd persisted as NUMERIC.
	var cost pgtype.Numeric
	require.NoError(t, testStore.pool.QueryRow(testCtx,
		"SELECT cost_usd FROM usage_log WHERE user_id=$1 ORDER BY id LIMIT 1", user).Scan(&cost))
	f, err := cost.Float64Value()
	require.NoError(t, err)
	assert.InDelta(t, 0.0015, f.Float64, 1e-9)
}

// ----- cascades -------------------------------------------------------------

func TestCascade_DeleteConversationRemovesMessagesAndBlocks(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user, conv := newConversation(t)
	m, err := testStore.CreateMessage(testCtx, db.CreateMessageParams{
		ConversationID: conv, UserID: user, Role: "user", Status: "complete",
	})
	require.NoError(t, err)
	_, err = testStore.CreateMessageBlock(testCtx, db.CreateMessageBlockParams{
		MessageID: m.ID, OrderIndex: 0, Type: "text", ContentText: textPtr("hi"),
	})
	require.NoError(t, err)

	_, err = testStore.pool.Exec(testCtx, "DELETE FROM conversations WHERE id=$1", conv)
	require.NoError(t, err)

	assert.Equal(t, 0, tableCount(t, "SELECT COUNT(*) FROM messages WHERE conversation_id=$1", conv))
	assert.Equal(t, 0, tableCount(t, "SELECT COUNT(*) FROM message_blocks WHERE message_id=$1", m.ID))
}

func TestCascade_HardDeleteUserRemovesEverything(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	user, conv := newConversation(t)
	m, err := testStore.CreateMessage(testCtx, db.CreateMessageParams{
		ConversationID: conv, UserID: user, Role: "user", Status: "complete",
	})
	require.NoError(t, err)
	_, err = testStore.CreateMessageBlock(testCtx, db.CreateMessageBlockParams{
		MessageID: m.ID, OrderIndex: 0, Type: "text", ContentText: textPtr("hi"),
	})
	require.NoError(t, err)
	_, err = testStore.CreateUpload(testCtx, db.CreateUploadParams{
		UserID: user, StorageKey: fmt.Sprintf("u/%s/x", user), ContentType: "image/jpeg", SizeBytes: 1,
	})
	require.NoError(t, err)
	require.NoError(t, testStore.InsertUsage(testCtx, db.InsertUsageParams{
		UserID: user, Endpoint: "/v1/messages", TokensIn: i4(1), TokensOut: i4(1),
	}))

	// Hard delete (ARCH §6.3) cascades through every FK.
	_, err = testStore.pool.Exec(testCtx, "DELETE FROM users WHERE id=$1", user)
	require.NoError(t, err)

	assert.Equal(t, 0, tableCount(t, "SELECT COUNT(*) FROM conversations WHERE user_id=$1", user))
	assert.Equal(t, 0, tableCount(t, "SELECT COUNT(*) FROM messages WHERE user_id=$1", user))
	assert.Equal(t, 0, tableCount(t, "SELECT COUNT(*) FROM message_blocks WHERE message_id=$1", m.ID))
	assert.Equal(t, 0, tableCount(t, "SELECT COUNT(*) FROM uploads WHERE user_id=$1", user))
	assert.Equal(t, 0, tableCount(t, "SELECT COUNT(*) FROM usage_log WHERE user_id=$1", user))
}
