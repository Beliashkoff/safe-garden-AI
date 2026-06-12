package auth

import "errors"

// Provider identifiers persisted in users.yandex_sub / users.vk_sub and used
// across the usecase layer.
const (
	ProviderYandex = "yandex"
	ProviderVK     = "vk"
)

// ExternalIdentity is what we trust after a completed provider code exchange.
//
// Subject is the provider-scoped stable user identifier (Yandex uid / VK
// user_id). Email is informational unless EmailVerified: Yandex returns the
// user's own Yandex mailbox (provider-verified), while VK ID does not document
// an email-confirmation guarantee, so VK emails are never treated as verified.
type ExternalIdentity struct {
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
}

// ErrOAuthRejected — the provider definitively rejected the request (invalid,
// expired or foreign authorization code / access token). Distinct from
// transport or 5xx failures, which are returned as plain wrapped errors and
// should surface as "try again later".
var ErrOAuthRejected = errors.New("auth: oauth request rejected by provider")
