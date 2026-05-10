package observability

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

var redactedKeys = map[string]struct{}{
	"email":         {},
	"otp":           {},
	"code":          {},
	"password":      {},
	"token":         {},
	"access_token":  {},
	"refresh_token": {},
	"id_token":      {},
	"secret":        {},
	"api_key":       {},
	"text":          {},
	"transcription": {},
	"message_text":  {},
}

const redactedValue = "[REDACTED]"

func NewLogger(env, level string) *slog.Logger {
	return newLoggerWithWriter(env, level, os.Stdout)
}

func newLoggerWithWriter(env, level string, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:       parseLevel(level),
		ReplaceAttr: redactPII,
	}
	var handler slog.Handler
	if env == "prod" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}
	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func redactPII(_ []string, a slog.Attr) slog.Attr {
	if _, ok := redactedKeys[a.Key]; ok {
		return slog.String(a.Key, redactedValue)
	}
	return a
}
