package chat

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

func TestValidateInput(t *testing.T) {
	_, err := validateInput(SendInput{})
	assert.ErrorIs(t, err, ErrEmptyContent)

	_, err = validateInput(SendInput{Blocks: []InputBlock{{Type: "text", Text: "   "}}})
	assert.ErrorIs(t, err, ErrEmptyContent)

	_, err = validateInput(SendInput{Blocks: []InputBlock{{Type: "image_ref", Text: ""}}})
	assert.ErrorIs(t, err, ErrUnsupportedBlock)

	text, err := validateInput(SendInput{Blocks: []InputBlock{{Type: "text", Text: "hello"}, {Type: "text", Text: " world"}}})
	require.NoError(t, err)
	assert.Equal(t, "hello world", text)

	big := make([]byte, maxTextBytes+1)
	for i := range big {
		big[i] = 'a'
	}
	_, err = validateInput(SendInput{Blocks: []InputBlock{{Type: "text", Text: string(big)}}})
	assert.ErrorIs(t, err, ErrTextTooLarge)
}

func TestUIDHash_DeterministicAndPeppered(t *testing.T) {
	u := uuid.New()
	h1 := uidHash(u, "pep")
	assert.Equal(t, h1, uidHash(u, "pep"), "deterministic")
	assert.NotEqual(t, h1, uidHash(u, "other"), "pepper changes the hash")
	assert.Len(t, h1, 64, "sha256 hex")
}

func TestCursor_RoundTrip(t *testing.T) {
	at := time.Unix(0, 1_700_000_000_123_456_789).UTC()
	id := uuid.New()
	gotAt, gotID, err := decodeCursor(encodeCursor(at, id))
	require.NoError(t, err)
	assert.True(t, at.Equal(gotAt))
	assert.Equal(t, id, gotID)

	_, _, err = decodeCursor("not-base64-!!")
	assert.ErrorIs(t, err, ErrBadCursor)
}

func TestClampLimit(t *testing.T) {
	assert.Equal(t, int32(defaultPageLimit), clampLimit(0))
	assert.Equal(t, int32(maxPageLimit), clampLimit(1000))
	assert.Equal(t, int32(10), clampLimit(10))
}

func msg(id uuid.UUID, role, status string, ts time.Time) db.Message {
	return db.Message{ID: id, Role: role, Status: status, CreatedAt: pgtype.Timestamptz{Time: ts, Valid: true}}
}

func textBlock(msgID uuid.UUID, text string) db.MessageBlock {
	return db.MessageBlock{MessageID: msgID, Type: "text", ContentText: pgtype.Text{String: text, Valid: text != ""}}
}

func TestBuildLLMMessages_FiltersAndMaps(t *testing.T) {
	u1, a1, aPending, empty := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	msgs := []db.Message{
		msg(u1, "user", "complete", time.Now()),
		msg(a1, "assistant", "complete", time.Now()),
		msg(aPending, "assistant", "pending", time.Now()), // skipped: not complete
		msg(empty, "user", "complete", time.Now()),        // skipped: no text block
	}
	blocks := map[uuid.UUID][]db.MessageBlock{
		u1: {textBlock(u1, "hi")},
		a1: {textBlock(a1, "hello")},
	}
	out := buildLLMMessages(msgs, blocks)
	require.Len(t, out, 2)
	assert.Equal(t, "user", out[0].Role)
	assert.Equal(t, "hi", out[0].Content[0].Text)
	assert.Equal(t, "assistant", out[1].Role)
	assert.Equal(t, "hello", out[1].Content[0].Text)
}

func TestPaginate(t *testing.T) {
	now := time.Now()
	rows := []db.Message{
		msg(uuid.New(), "user", "complete", now.Add(3*time.Second)),
		msg(uuid.New(), "user", "complete", now.Add(2*time.Second)),
		msg(uuid.New(), "user", "complete", now.Add(1*time.Second)), // extra (limit+1 probe)
	}
	page, next := paginate(rows, 2)
	assert.Len(t, page, 2)
	assert.NotEmpty(t, next, "more pages exist → cursor set")

	page2, next2 := paginate(rows[:2], 2)
	assert.Len(t, page2, 2)
	assert.Empty(t, next2, "exactly a page, no probe → no cursor")
}
