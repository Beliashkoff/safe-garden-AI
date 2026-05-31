package audio

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockTranscriber_ReturnsFixture(t *testing.T) {
	t.Parallel()
	res, err := NewMockTranscriber().Transcribe(context.Background(), []byte("ignored"), "ru-RU")
	require.NoError(t, err)
	assert.NotEmpty(t, res.Text)
	assert.Equal(t, int64(1000), res.DurationMs)
}

func TestMockTranscriber_RespectsCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewMockTranscriber().Transcribe(ctx, nil, "")
	require.Error(t, err)
}

func TestNew_SelectsProvider(t *testing.T) {
	t.Parallel()

	tr, err := New(&Config{Kind: "mock"})
	require.NoError(t, err)
	_, ok := tr.(*MockTranscriber)
	assert.True(t, ok, "mock kind should yield *MockTranscriber")

	tr, err = New(&Config{Kind: ""})
	require.NoError(t, err)
	_, ok = tr.(*MockTranscriber)
	assert.True(t, ok, "empty kind should default to *MockTranscriber")

	tr, err = New(&Config{Kind: "gigachat"})
	require.NoError(t, err)
	_, ok = tr.(*GigaChatTranscriber)
	assert.True(t, ok, "gigachat kind should yield *GigaChatTranscriber")

	_, err = New(&Config{Kind: "bogus"})
	require.Error(t, err)

	// speechkit without an API key must fail fast.
	_, err = New(&Config{Kind: "speechkit", SpeechKitEndpoint: "stt.api.cloud.yandex.net:443"})
	require.Error(t, err)
}

func TestGigaChat_NotImplemented(t *testing.T) {
	t.Parallel()
	_, err := NewGigaChatTranscriber(&Config{}).Transcribe(context.Background(), nil, "")
	require.ErrorIs(t, err, ErrNotImplemented)
}
