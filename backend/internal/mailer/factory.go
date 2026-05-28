package mailer

import "log/slog"

// New selects a Mailer implementation from the SMTP config. An empty Host means
// "no SMTP" → the log fallback (suitable for tests / bare local runs). Otherwise
// the net/smtp implementation is returned (MailHog in dev, Yandex 360 in prod).
func New(cfg SMTPConfig, logger *slog.Logger) Mailer {
	if cfg.Host == "" {
		return &logMailer{logger: logger}
	}
	return newSMTPMailer(cfg)
}
