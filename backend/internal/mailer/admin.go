package mailer

import (
	"bytes"
	"context"
	"fmt"
)

// Admin code purposes. Mirror the admin_codes.purpose CHECK constraint.
const (
	AdminCodePurposeSetup = "setup"
	AdminCodePurposeReset = "reset"
)

// AdminMailer delivers one-time codes for the admin panel (first-time setup
// and password reset). Separate from Mailer so existing implementations and
// mocks of the user-facing OTP flow are untouched. Implementations must never
// log the code.
type AdminMailer interface {
	SendAdminCode(ctx context.Context, to, code, purpose string) error
}

type adminCodeData struct {
	Intro      string
	Code       string
	TTLMinutes int
}

func adminSubject(purpose string) string {
	if purpose == AdminCodePurposeReset {
		return "Код восстановления пароля админ-панели Safe Garden AI"
	}
	return "Код настройки админ-панели Safe Garden AI"
}

func adminIntro(purpose string) string {
	if purpose == AdminCodePurposeReset {
		return "Ваш код для восстановления пароля админ-панели"
	}
	return "Ваш код для первичной настройки админ-панели"
}

func renderAdminCode(purpose, code string) (text, html string, err error) {
	data := adminCodeData{Intro: adminIntro(purpose), Code: code, TTLMinutes: otpTTLMinutes}
	var tb, hb bytes.Buffer
	if err := textTemplates.ExecuteTemplate(&tb, "admin_code_ru.txt", data); err != nil {
		return "", "", fmt.Errorf("mailer: render admin text: %w", err)
	}
	if err := htmlTemplates.ExecuteTemplate(&hb, "admin_code_ru.html", data); err != nil {
		return "", "", fmt.Errorf("mailer: render admin html: %w", err)
	}
	return tb.String(), hb.String(), nil
}

func (m *smtpMailer) SendAdminCode(ctx context.Context, to, code, purpose string) error {
	text, html, err := renderAdminCode(purpose, code)
	if err != nil {
		return err
	}
	msg, err := buildMIME(m.from, to, adminSubject(purpose), text, html, m.now())
	if err != nil {
		return err
	}
	return m.send(ctx, to, msg)
}

func (m *logMailer) SendAdminCode(ctx context.Context, to, _ /*code*/, purpose string) error {
	m.logger.InfoContext(ctx, "admin code email suppressed (no SMTP configured)",
		"email", to, "purpose", purpose)
	return nil
}
