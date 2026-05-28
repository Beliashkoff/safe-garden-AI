package mailer

import (
	"context"
	"log/slog"
)

// logMailer is the no-SMTP fallback used when no SMTP host is configured. It
// records that an OTP was "sent" without ever logging the code itself (PII /
// secret — CLAUDE.md §3). Intended for unit tests and bare local runs without
// MailHog; real dev uses the SMTP mailer against MailHog.
type logMailer struct {
	logger *slog.Logger
}

func (m *logMailer) SendOTP(ctx context.Context, to, _ /*code*/, locale string) error {
	// Key "email" is redacted by the slog PII replacer; the code is never logged.
	m.logger.InfoContext(ctx, "otp email suppressed (no SMTP configured)",
		"email", to, "locale", locale)
	return nil
}
