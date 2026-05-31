package audio

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// FFmpegConverter shells out to ffmpeg/ffprobe (ROADMAP 4.1). It is the only
// component that requires the binaries to be present; they are installed in the
// API Docker image.
type FFmpegConverter struct {
	ffmpegPath  string
	ffprobePath string
	maxDuration int64 // ms; sources longer than this are rejected with ErrTooLong
}

func NewConverter(ffmpegPath, ffprobePath string) *FFmpegConverter {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}
	return &FFmpegConverter{ffmpegPath: ffmpegPath, ffprobePath: ffprobePath, maxDuration: MaxDurationMs}
}

var _ Converter = (*FFmpegConverter)(nil)

// ToOggOpus converts input audio to OggOpus 16 kHz mono and returns the source
// duration. contentType is accepted for interface stability; ffmpeg detects the
// real format from the bytes, so it is not used to pick a demuxer.
func (c *FFmpegConverter) ToOggOpus(ctx context.Context, input []byte, _ string) ([]byte, int64, error) {
	if len(input) == 0 {
		return nil, 0, fmt.Errorf("audio.converter: empty input")
	}

	// The mp4/m4a demuxer needs a seekable source to find the moov atom, which
	// mobile recordings often place at the end of the file. A temp file gives
	// ffmpeg/ffprobe random access; a stdin pipe would not.
	src, err := os.CreateTemp("", "sga-audio-*")
	if err != nil {
		return nil, 0, fmt.Errorf("audio.converter: temp file: %w", err)
	}
	defer func() { _ = os.Remove(src.Name()) }()
	if _, err := src.Write(input); err != nil {
		_ = src.Close()
		return nil, 0, fmt.Errorf("audio.converter: write temp: %w", err)
	}
	if err := src.Close(); err != nil {
		return nil, 0, fmt.Errorf("audio.converter: close temp: %w", err)
	}

	durationMs, err := c.probeDurationMs(ctx, src.Name())
	if err != nil {
		return nil, 0, err
	}
	if durationMs > c.maxDuration {
		return nil, durationMs, ErrTooLong
	}

	out, err := c.convert(ctx, src.Name())
	if err != nil {
		return nil, durationMs, err
	}
	return out, durationMs, nil
}

func (c *FFmpegConverter) convert(ctx context.Context, srcPath string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.ffmpegPath,
		"-hide_banner", "-nostdin",
		"-i", srcPath,
		"-c:a", "libopus", "-ar", "16000", "-ac", "1",
		"-f", "ogg", "pipe:1",
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("audio.converter: ffmpeg: %w: %s", err, tailStderr(stderr.String()))
	}
	if stdout.Len() == 0 {
		return nil, fmt.Errorf("audio.converter: ffmpeg produced no output: %s", tailStderr(stderr.String()))
	}
	return stdout.Bytes(), nil
}

func (c *FFmpegConverter) probeDurationMs(ctx context.Context, srcPath string) (int64, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		srcPath,
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("audio.converter: ffprobe: %w: %s", err, tailStderr(stderr.String()))
	}
	raw := strings.TrimSpace(stdout.String())
	secs, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("audio.converter: parse duration %q: %w", raw, err)
	}
	return int64(secs * 1000), nil
}

// tailStderr returns the last chunk of ffmpeg/ffprobe diagnostics. Their stderr
// is log text only (never the audio bytes), so it is safe to surface in errors.
func tailStderr(s string) string {
	const max = 512
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}
