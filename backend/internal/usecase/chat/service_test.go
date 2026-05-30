package chat

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

func TestValidateInput(t *testing.T) {
	uid := uuid.New()
	pfx := "u/" + uid.String() + "/"

	_, err := validateInput(uid, SendInput{})
	assert.ErrorIs(t, err, ErrEmptyContent)

	_, err = validateInput(uid, SendInput{Blocks: []InputBlock{{Type: "text", Text: "   "}}})
	assert.ErrorIs(t, err, ErrEmptyContent)

	// image_ref without a key, or with a foreign-owner prefix → not found.
	_, err = validateInput(uid, SendInput{Blocks: []InputBlock{{Type: "image_ref"}}})
	assert.ErrorIs(t, err, ErrUploadNotFound)
	_, err = validateInput(uid, SendInput{Blocks: []InputBlock{
		{Type: "image_ref", StorageKey: "u/" + uuid.New().String() + "/img/x.jpg"},
	}})
	assert.ErrorIs(t, err, ErrUploadNotFound)

	// audio_ref is Stage 4 → unsupported.
	_, err = validateInput(uid, SendInput{Blocks: []InputBlock{{Type: "audio_ref", StorageKey: pfx + "a/x.m4a"}}})
	assert.ErrorIs(t, err, ErrUnsupportedBlock)

	// text + image → accepted and ordered.
	blocks, err := validateInput(uid, SendInput{Blocks: []InputBlock{
		{Type: "text", Text: "hello"},
		{Type: "image_ref", StorageKey: pfx + "img/a.jpg"},
	}})
	require.NoError(t, err)
	require.Len(t, blocks, 2)
	assert.Equal(t, "text", blocks[0].kind)
	assert.Equal(t, "hello", blocks[0].text)
	assert.Equal(t, "image", blocks[1].kind)
	assert.Equal(t, pfx+"img/a.jpg", blocks[1].storageKey)

	// Image-only message is allowed.
	blocks, err = validateInput(uid, SendInput{Blocks: []InputBlock{{Type: "image_ref", StorageKey: pfx + "img/a.jpg"}}})
	require.NoError(t, err)
	require.Len(t, blocks, 1)

	// Too many images.
	imgs := make([]InputBlock, maxImagesPerMessage+1)
	for i := range imgs {
		imgs[i] = InputBlock{Type: "image_ref", StorageKey: pfx + "img/" + string(rune('a'+i)) + ".jpg"}
	}
	_, err = validateInput(uid, SendInput{Blocks: imgs})
	assert.ErrorIs(t, err, ErrUnsupportedBlock)

	// Text over the cap.
	big := make([]byte, maxTextBytes+1)
	for i := range big {
		big[i] = 'a'
	}
	_, err = validateInput(uid, SendInput{Blocks: []InputBlock{{Type: "text", Text: string(big)}}})
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

func imageBlock(msgID uuid.UUID, key string) db.MessageBlock {
	return db.MessageBlock{MessageID: msgID, Type: "image", StorageKey: pgtype.Text{String: key, Valid: true}}
}

func TestAssembleMessages_FiltersAndMaps(t *testing.T) {
	u1, a1, aPending, empty := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	msgs := []db.Message{
		msg(u1, "user", "complete", time.Now()),
		msg(a1, "assistant", "complete", time.Now()),
		msg(aPending, "assistant", "pending", time.Now()), // skipped: not complete
		msg(empty, "user", "complete", time.Now()),        // skipped: no usable block
	}
	blocks := map[uuid.UUID][]db.MessageBlock{
		u1: {textBlock(u1, "hi")},
		a1: {textBlock(a1, "hello")},
	}
	out := assembleMessages(msgs, blocks, nil)
	require.Len(t, out, 2)
	assert.Equal(t, "user", out[0].Role)
	assert.Equal(t, "hi", out[0].Content[0].Text)
	assert.Equal(t, "assistant", out[1].Role)
	assert.Equal(t, "hello", out[1].Content[0].Text)
}

func TestAssembleMessages_ImagesHydratedOrPlaceholder(t *testing.T) {
	u := uuid.New()
	msgs := []db.Message{msg(u, "user", "complete", time.Now())}
	blocks := map[uuid.UUID][]db.MessageBlock{
		u: {textBlock(u, "look"), imageBlock(u, "u/x/img/a.jpg"), imageBlock(u, "u/x/img/b.jpg")},
	}
	hydrated := map[string]llm.MessageBlock{
		"u/x/img/a.jpg": {Type: "image", MediaB64: "QQ==", MediaType: "image/jpeg"},
	}
	out := assembleMessages(msgs, blocks, hydrated)
	require.Len(t, out, 1)
	require.Len(t, out[0].Content, 3)
	assert.Equal(t, "text", out[0].Content[0].Type)
	assert.Equal(t, "image", out[0].Content[1].Type) // hydrated → base64 image
	assert.Equal(t, "image/jpeg", out[0].Content[1].MediaType)
	assert.Equal(t, "text", out[0].Content[2].Type) // beyond cap → placeholder
	assert.Equal(t, imagePlaceholder, out[0].Content[2].Text)
}

func TestSelectRecentImages_NewestFirstCapped(t *testing.T) {
	now := time.Now()
	m1, m2, m3 := uuid.New(), uuid.New(), uuid.New()
	msgs := []db.Message{ // chronological
		msg(m1, "user", "complete", now.Add(1*time.Second)),
		msg(m2, "user", "complete", now.Add(2*time.Second)),
		msg(m3, "user", "complete", now.Add(3*time.Second)),
	}
	blocks := map[uuid.UUID][]db.MessageBlock{
		m1: {imageBlock(m1, "k1")},
		m2: {imageBlock(m2, "k2")},
		m3: {imageBlock(m3, "k3")},
	}
	assert.Equal(t, []string{"k3", "k2"}, selectRecentImages(msgs, blocks, 2))
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
