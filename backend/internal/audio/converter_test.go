package audio

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireFFmpeg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed; skipping converter test")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed; skipping converter test")
	}
}

// genAAC synthesizes `seconds` of a sine tone encoded as AAC in an m4a
// container, the format the mobile client records.
func genAAC(t *testing.T, seconds int) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	f, err := os.CreateTemp("", "tone-*.m4a")
	require.NoError(t, err)
	_ = f.Close()
	defer func() { _ = os.Remove(f.Name()) }()

	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-hide_banner", "-nostdin",
		"-f", "lavfi", "-i", fmt.Sprintf("sine=frequency=440:duration=%d", seconds),
		"-c:a", "aac", f.Name(),
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	data, err := os.ReadFile(f.Name())
	require.NoError(t, err)
	require.NotEmpty(t, data)
	return data
}

func TestConverter_ToOggOpus(t *testing.T) {
	requireFFmpeg(t)
	input := genAAC(t, 2)

	out, dur, err := NewConverter("", "").ToOggOpus(context.Background(), input, "audio/mp4")
	require.NoError(t, err)
	assert.NotEmpty(t, out)
	assert.Greater(t, dur, int64(1000))
	assert.Less(t, dur, int64(4000))
	require.GreaterOrEqual(t, len(out), 4)
	assert.Equal(t, "OggS", string(out[:4]), "output should be an Ogg stream")
}

func TestConverter_RejectsTooLong(t *testing.T) {
	requireFFmpeg(t)
	input := genAAC(t, 61)

	_, dur, err := NewConverter("", "").ToOggOpus(context.Background(), input, "audio/mp4")
	require.ErrorIs(t, err, ErrTooLong)
	assert.Greater(t, dur, MaxDurationMs)
}

func TestConverter_RejectsEmpty(t *testing.T) {
	t.Parallel()
	_, _, err := NewConverter("", "").ToOggOpus(context.Background(), nil, "audio/mp4")
	require.Error(t, err)
}
