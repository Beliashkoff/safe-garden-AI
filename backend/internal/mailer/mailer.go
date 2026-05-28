// Package mailer delivers transactional email — currently only the sign-in OTP.
//
// The implementation uses the standard library net/smtp (no third-party SMTP
// dependency): the dev target is the docker-compose MailHog (plaintext,
// no auth) and prod is Yandex 360 (implicit TLS on :465, AUTH LOGIN). The
// Mailer interface is intentionally tiny so the usecase layer depends on a
// behaviour, not on SMTP details.
package mailer

import "context"

// Mailer sends transactional messages. locale selects the template language
// ("ru" default; "en" supported). Implementations must never log the OTP code.
type Mailer interface {
	SendOTP(ctx context.Context, to, code, locale string) error
}
