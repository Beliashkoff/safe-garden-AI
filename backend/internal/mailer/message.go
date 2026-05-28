package mailer

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
	texttemplate "text/template"
	"time"
)

//go:embed templates/*
var templatesFS embed.FS

// otpTTLMinutes mirrors the OTP lifetime enforced in storage (10 min, ARCH
// §8.1). Shown to the user in the email body.
const otpTTLMinutes = 10

var (
	htmlTemplates = template.Must(template.ParseFS(templatesFS, "templates/*.html"))
	textTemplates = texttemplate.Must(texttemplate.ParseFS(templatesFS, "templates/*.txt"))
)

type otpData struct {
	Code       string
	TTLMinutes int
}

// normalizeLocale collapses anything to the two supported template languages.
func normalizeLocale(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "en") {
		return "en"
	}
	return "ru"
}

func subjectFor(locale string) string {
	if normalizeLocale(locale) == "en" {
		return "Your Safe Garden AI sign-in code"
	}
	return "Код для входа в Safe Garden AI"
}

func renderOTP(locale, code string) (text, html string, err error) {
	loc := normalizeLocale(locale)
	data := otpData{Code: code, TTLMinutes: otpTTLMinutes}
	var tb, hb bytes.Buffer
	if err := textTemplates.ExecuteTemplate(&tb, "otp_"+loc+".txt", data); err != nil {
		return "", "", fmt.Errorf("mailer: render text: %w", err)
	}
	if err := htmlTemplates.ExecuteTemplate(&hb, "otp_"+loc+".html", data); err != nil {
		return "", "", fmt.Errorf("mailer: render html: %w", err)
	}
	return tb.String(), hb.String(), nil
}

// buildMIME assembles a multipart/alternative RFC 5322 message with a
// quoted-printable plain and HTML part. Subject is RFC 2047 encoded so Cyrillic
// survives transport. Returns the raw bytes ready for the SMTP DATA command.
func buildMIME(from mail.Address, to, subject, text, html string, now time.Time) ([]byte, error) {
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	boundary := mw.Boundary()

	if err := writePart(mw, "text/plain; charset=utf-8", text); err != nil {
		return nil, err
	}
	if err := writePart(mw, "text/html; charset=utf-8", html); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("mailer: close multipart: %w", err)
	}

	var msg bytes.Buffer
	writeHeader(&msg, "From", from.String())
	writeHeader(&msg, "To", (&mail.Address{Address: to}).String())
	writeHeader(&msg, "Subject", mime.QEncoding.Encode("utf-8", subject))
	writeHeader(&msg, "Date", now.Format(time.RFC1123Z))
	writeHeader(&msg, "Message-ID", messageID(from.Address))
	writeHeader(&msg, "MIME-Version", "1.0")
	writeHeader(&msg, "Content-Type", "multipart/alternative; boundary=\""+boundary+"\"")
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())
	return msg.Bytes(), nil
}

func writePart(mw *multipart.Writer, contentType, content string) error {
	pw, err := mw.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {contentType},
		"Content-Transfer-Encoding": {"quoted-printable"},
	})
	if err != nil {
		return fmt.Errorf("mailer: create part: %w", err)
	}
	return writeQP(pw, content)
}

func writeQP(w io.Writer, s string) error {
	qp := quotedprintable.NewWriter(w)
	if _, err := qp.Write([]byte(s)); err != nil {
		return fmt.Errorf("mailer: qp write: %w", err)
	}
	return qp.Close()
}

func writeHeader(buf *bytes.Buffer, key, val string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(val)
	buf.WriteString("\r\n")
}

func messageID(fromAddr string) string {
	domain := "localhost"
	if i := strings.LastIndex(fromAddr, "@"); i >= 0 && i+1 < len(fromAddr) {
		domain = fromAddr[i+1:]
	}
	var b [16]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("<%s@%s>", hex.EncodeToString(b[:]), domain)
}
