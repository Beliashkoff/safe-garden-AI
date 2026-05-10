package observability

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactPII_SensitiveKeysAreRedacted(t *testing.T) {
	cases := []struct {
		key   string
		value string
	}{
		{"email", "alice@example.com"},
		{"otp", "123456"},
		{"code", "987654"},
		{"password", "hunter2"},
		{"token", "abc.def.ghi"},
		{"access_token", "eyJ..."},
		{"refresh_token", "rtk_..."},
		{"id_token", "id.token.value"},
		{"secret", "topsecret"},
		{"api_key", "sk-xxxx"},
		{"text", "user wrote this private message"},
		{"transcription", "transcribed voice content"},
		{"message_text", "hello there"},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			var buf bytes.Buffer
			logger := newLoggerWithWriter("prod", "debug", &buf)

			logger.Info("event", tc.key, tc.value)

			out := buf.String()
			assert.NotContains(t, out, tc.value, "raw value leaked into logs")
			assert.Contains(t, out, redactedValue, "redacted marker missing")
		})
	}
}

func TestRedactPII_NonSensitiveKeysPassThrough(t *testing.T) {
	var buf bytes.Buffer
	logger := newLoggerWithWriter("prod", "debug", &buf)

	logger.Info("event",
		"request_id", "req-abc",
		"user_id_hash", "hash-xyz",
		"endpoint", "/healthz",
		"status", 200,
	)

	out := buf.String()
	assert.Contains(t, out, "req-abc")
	assert.Contains(t, out, "hash-xyz")
	assert.Contains(t, out, "/healthz")
	assert.NotContains(t, out, redactedValue)
}

func TestParseLevel(t *testing.T) {
	assert.Equal(t, slog.LevelDebug, parseLevel("debug"))
	assert.Equal(t, slog.LevelInfo, parseLevel("info"))
	assert.Equal(t, slog.LevelWarn, parseLevel("warn"))
	assert.Equal(t, slog.LevelWarn, parseLevel("WARNING"))
	assert.Equal(t, slog.LevelError, parseLevel("error"))
	assert.Equal(t, slog.LevelInfo, parseLevel(""), "default to info")
	assert.Equal(t, slog.LevelInfo, parseLevel("nonsense"), "unknown -> info")
}

func TestNewLogger_DevUsesText(t *testing.T) {
	var buf bytes.Buffer
	logger := newLoggerWithWriter("dev", "debug", &buf)
	logger.Info("hello", "k", "v")

	// Text handler renders k=v pairs without JSON braces.
	out := buf.String()
	assert.True(t, strings.Contains(out, "k=v"), "expected text format, got %q", out)
	assert.False(t, strings.HasPrefix(strings.TrimSpace(out), "{"), "should not be JSON in dev")
}

func TestNewLogger_ProdUsesJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := newLoggerWithWriter("prod", "info", &buf)
	logger.Info("hello", "k", "v")

	out := strings.TrimSpace(buf.String())
	assert.True(t, strings.HasPrefix(out, "{"), "expected JSON in prod, got %q", out)
	assert.True(t, strings.HasSuffix(out, "}"), "expected JSON in prod, got %q", out)
}
