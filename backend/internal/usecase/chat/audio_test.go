package chat

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/audio"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

// --- fakes ---

type fakeMedia struct {
	data        []byte
	contentType string
	err         error
	gotKey      string
	gotMax      int64
}

func (f *fakeMedia) Get(ctx context.Context, key string) ([]byte, string, error) {
	return f.GetLimited(ctx, key, 0)
}

func (f *fakeMedia) GetLimited(_ context.Context, key string, max int64) ([]byte, string, error) {
	f.gotKey, f.gotMax = key, max
	if f.err != nil {
		return nil, "", f.err
	}
	return f.data, f.contentType, nil
}

type fakeConverter struct {
	ogg        []byte
	durationMs int64
	err        error
	gotCT      string
}

func (f *fakeConverter) ToOggOpus(_ context.Context, _ []byte, ct string) ([]byte, int64, error) {
	f.gotCT = ct
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.ogg, f.durationMs, nil
}

type fakeTranscriber struct {
	res     audio.Result
	err     error
	gotOgg  []byte
	gotLang string
}

func (f *fakeTranscriber) Transcribe(_ context.Context, ogg []byte, lang string) (audio.Result, error) {
	f.gotOgg, f.gotLang = ogg, lang
	if f.err != nil {
		return audio.Result{}, f.err
	}
	return f.res, nil
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func audioSvc(media *fakeMedia, conv *fakeConverter, stt *fakeTranscriber) *Service {
	return &Service{images: media, audioConv: conv, stt: stt, lang: "ru-RU", logger: discardLogger()}
}

// --- transcribeAudio ---

func TestTranscribeAudio_Success(t *testing.T) {
	media := &fakeMedia{data: []byte("m4a-bytes"), contentType: "audio/m4a"}
	conv := &fakeConverter{ogg: []byte("ogg-bytes"), durationMs: 4200}
	stt := &fakeTranscriber{res: audio.Result{Text: "вянут помидоры", DurationMs: 9999}}
	s := audioSvc(media, conv, stt)

	blocks := []validatedBlock{
		{kind: "text", text: "hi"},
		{kind: "audio", storageKey: "u/x/audio/a.m4a"},
	}
	require.NoError(t, s.transcribeAudio(context.Background(), blocks))

	assert.Equal(t, "вянут помидоры", blocks[1].transcriptText)
	assert.Equal(t, int64(4200), blocks[1].durationMs, "converter duration is authoritative")
	assert.Equal(t, "u/x/audio/a.m4a", media.gotKey)
	assert.Equal(t, int64(maxAudioReadBytes), media.gotMax)
	assert.Equal(t, "audio/m4a", conv.gotCT)
	assert.Equal(t, []byte("ogg-bytes"), stt.gotOgg)
	assert.Equal(t, "ru-RU", stt.gotLang)
	assert.Empty(t, blocks[0].transcriptText, "text block untouched")
}

func TestTranscribeAudio_TooLong(t *testing.T) {
	s := audioSvc(
		&fakeMedia{data: []byte("x"), contentType: "audio/m4a"},
		&fakeConverter{err: audio.ErrTooLong},
		&fakeTranscriber{},
	)
	err := s.transcribeAudio(context.Background(), []validatedBlock{{kind: "audio", storageKey: "k"}})
	assert.ErrorIs(t, err, ErrAudioTooLong)
}

func TestTranscribeAudio_Empty(t *testing.T) {
	s := audioSvc(
		&fakeMedia{data: []byte("x"), contentType: "audio/m4a"},
		&fakeConverter{ogg: []byte("o"), durationMs: 1000},
		&fakeTranscriber{err: audio.ErrEmptyResult},
	)
	err := s.transcribeAudio(context.Background(), []validatedBlock{{kind: "audio", storageKey: "k"}})
	assert.ErrorIs(t, err, ErrTranscriptionEmpty)
}

func TestTranscribeAudio_FetchError(t *testing.T) {
	s := audioSvc(&fakeMedia{err: errors.New("boom")}, &fakeConverter{}, &fakeTranscriber{})
	err := s.transcribeAudio(context.Background(), []validatedBlock{{kind: "audio", storageKey: "k"}})
	assert.ErrorIs(t, err, ErrTranscriptionFailed)
}

func TestTranscribeAudio_ConvertError(t *testing.T) {
	s := audioSvc(
		&fakeMedia{data: []byte("x"), contentType: "audio/m4a"},
		&fakeConverter{err: errors.New("ffmpeg boom")},
		&fakeTranscriber{},
	)
	err := s.transcribeAudio(context.Background(), []validatedBlock{{kind: "audio", storageKey: "k"}})
	assert.ErrorIs(t, err, ErrTranscriptionFailed)
}

func TestTranscribeAudio_NoAudioIsNoop(t *testing.T) {
	media := &fakeMedia{err: errors.New("must not be read")}
	s := audioSvc(media, &fakeConverter{}, &fakeTranscriber{})
	require.NoError(t, s.transcribeAudio(context.Background(), []validatedBlock{{kind: "text", text: "hi"}}))
	assert.Empty(t, media.gotKey, "no object should be fetched")
}

// --- history + read projection ---

func audioDBBlock(msgID uuid.UUID, key string) db.MessageBlock {
	return db.MessageBlock{MessageID: msgID, Type: "audio", StorageKey: pgtype.Text{String: key, Valid: true}}
}

func transcriptionDBBlock(msgID uuid.UUID, text string, durationMs int64) db.MessageBlock {
	return db.MessageBlock{
		MessageID:   msgID,
		Type:        "transcription",
		ContentText: pgtype.Text{String: text, Valid: true},
		Metadata:    durationMeta(durationMs),
	}
}

func TestAssembleMessages_TranscriptionMarkedAudioSkipped(t *testing.T) {
	u := uuid.New()
	msgs := []db.Message{msg(u, "user", "complete", time.Now())}
	blocks := map[uuid.UUID][]db.MessageBlock{
		u: {audioDBBlock(u, "u/x/audio/a.m4a"), transcriptionDBBlock(u, "вянут помидоры", 4200)},
	}
	out := assembleMessages(msgs, blocks, nil)
	require.Len(t, out, 1)
	require.Len(t, out[0].Content, 1, "audio skipped, transcription → one text block")
	assert.Equal(t, "text", out[0].Content[0].Type)
	assert.Equal(t, voicePrefix+"вянут помидоры", out[0].Content[0].Text)
}

func TestToMessageView_AudioAndTranscription(t *testing.T) {
	u := uuid.New()
	m := msg(u, "user", "complete", time.Now())
	blocks := []db.MessageBlock{
		audioDBBlock(u, "u/x/audio/a.m4a"),
		transcriptionDBBlock(u, "текст", 4200),
	}
	v := toMessageView(m, blocks)
	require.Len(t, v.Content, 2)
	assert.Equal(t, "audio", v.Content[0].Type)
	assert.Equal(t, "u/x/audio/a.m4a", v.Content[0].StorageKey)
	assert.Equal(t, "transcription", v.Content[1].Type)
	assert.Equal(t, "текст", v.Content[1].Text)
	assert.Equal(t, int64(4200), v.Content[1].DurationMs)
}

func TestDurationMetaRoundTrip(t *testing.T) {
	assert.Equal(t, int64(0), durationFromMeta(nil))
	assert.Equal(t, int64(0), durationFromMeta([]byte("not json")))
	assert.Equal(t, int64(4200), durationFromMeta(durationMeta(4200)))
}
