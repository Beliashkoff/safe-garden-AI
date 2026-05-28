package mailer

import (
	"mime"
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderOTP_LocaleSelection(t *testing.T) {
	ruText, ruHTML, err := renderOTP("ru", "123456")
	require.NoError(t, err)
	assert.Contains(t, ruText, "123456")
	assert.Contains(t, ruText, "Safe Garden AI")
	assert.Contains(t, ruHTML, "123456")

	enText, enHTML, err := renderOTP("en-US", "654321")
	require.NoError(t, err)
	assert.Contains(t, enText, "654321")
	assert.Contains(t, enText, "sign-in code")
	assert.Contains(t, enHTML, "654321")
}

func TestRenderOTP_UnknownLocaleDefaultsRU(t *testing.T) {
	text, _, err := renderOTP("fr", "111222")
	require.NoError(t, err)
	assert.Contains(t, text, "код", "unknown locale should fall back to Russian")
}

func TestBuildMIME_WellFormed(t *testing.T) {
	from := mail.Address{Name: "Safe Garden AI", Address: "noreply@example.com"}
	text, html, err := renderOTP("ru", "424242")
	require.NoError(t, err)

	raw, err := buildMIME(from, "user@example.org", subjectFor("ru"), text, html,
		time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	require.NoError(t, err)

	assert.Equal(t, "<user@example.org>", msg.Header.Get("To"))
	assert.True(t, strings.HasPrefix(msg.Header.Get("Content-Type"), "multipart/alternative"),
		"got %q", msg.Header.Get("Content-Type"))

	dec := new(mime.WordDecoder)
	subject, err := dec.DecodeHeader(msg.Header.Get("Subject"))
	require.NoError(t, err)
	assert.Equal(t, "Код для входа в Safe Garden AI", subject)
}
