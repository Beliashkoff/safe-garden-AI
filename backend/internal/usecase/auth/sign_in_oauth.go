package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

const (
	// oauthStateTTL — how long a started attempt stays valid. Yandex/VK codes
	// themselves live 10 minutes; matching that keeps the windows aligned.
	oauthStateTTL = otpTTL
	// maxActiveOAuthStatesPerIP caps pending attempts per client IP so /start
	// cannot be used to flood the oauth_states table.
	maxActiveOAuthStatesPerIP = 20
)

// OAuthStart is handed to the mobile client to run the provider UI. The PKCE
// code_verifier never leaves the backend — only the derived S256 challenge.
type OAuthStart struct {
	// State doubles as the attempt handle; it comes back in /complete.
	State string
	// CodeChallenge is consumed by the VK ID SDK (confidential flow). For
	// Yandex it is already embedded in AuthURL.
	CodeChallenge string
	// AuthURL — full oauth.yandex.ru/authorize URL for the system browser.
	// Empty for VK (the native SDK builds its own UI).
	AuthURL string
}

// StartYandex begins a Yandex ID sign-in attempt.
func (s *Service) StartYandex(ctx context.Context, dev DeviceMeta) (OAuthStart, error) {
	start, err := s.startOAuth(ctx, authpkg.ProviderYandex, dev)
	if err != nil {
		return OAuthStart{}, err
	}
	start.AuthURL = s.yandex.AuthorizeURL(start.State, start.CodeChallenge)
	return start, nil
}

// StartVK begins a VK ID sign-in attempt.
func (s *Service) StartVK(ctx context.Context, dev DeviceMeta) (OAuthStart, error) {
	return s.startOAuth(ctx, authpkg.ProviderVK, dev)
}

func (s *Service) startOAuth(ctx context.Context, provider string, dev DeviceMeta) (OAuthStart, error) {
	// Best-effort sweep keeps the table from accumulating abandoned attempts.
	if err := s.store.DeleteExpiredOAuthStates(ctx); err != nil {
		s.logger.WarnContext(ctx, "oauth states sweep failed", "err", err.Error())
	}

	ip := ipText(dev)
	if ip.Valid {
		n, err := s.store.CountActiveOAuthStatesByIP(ctx, ip)
		if err != nil {
			return OAuthStart{}, fmt.Errorf("auth: count oauth states: %w", err)
		}
		if n >= maxActiveOAuthStatesPerIP {
			return OAuthStart{}, ErrRateLimited
		}
	}

	state, stateHash, err := authpkg.NewOAuthState()
	if err != nil {
		return OAuthStart{}, fmt.Errorf("auth: new oauth state: %w", err)
	}
	verifier, challenge, err := authpkg.NewPKCEVerifier()
	if err != nil {
		return OAuthStart{}, fmt.Errorf("auth: new pkce verifier: %w", err)
	}
	if err := s.store.CreateOAuthState(ctx, db.CreateOAuthStateParams{
		StateHash:    stateHash,
		Provider:     provider,
		CodeVerifier: verifier,
		Ip:           ip,
		ExpiresAt:    timestamptz(s.now().Add(oauthStateTTL)),
	}); err != nil {
		return OAuthStart{}, fmt.Errorf("auth: persist oauth state: %w", err)
	}
	return OAuthStart{State: state, CodeChallenge: challenge}, nil
}

// CompleteYandex finishes the Yandex ID flow: consume the one-time state,
// exchange the code (PKCE verifier + client_secret stay server-side), verify
// the identity JWT, and sign the user in.
func (s *Service) CompleteYandex(ctx context.Context, code, state string, dev DeviceMeta) (AuthResult, error) {
	row, err := s.consumeState(ctx, authpkg.ProviderYandex, state)
	if err != nil {
		return AuthResult{}, err
	}
	id, err := s.yandex.SignIn(ctx, code, row.CodeVerifier)
	if err != nil {
		return AuthResult{}, mapProviderErr(err)
	}
	return s.signInExternal(ctx, id, "sign_in_yandex", dev)
}

// CompleteVK finishes the VK ID flow. deviceID is the value VK issued next to
// the code; it is required by the token exchange.
func (s *Service) CompleteVK(ctx context.Context, code, state, deviceID string, dev DeviceMeta) (AuthResult, error) {
	row, err := s.consumeState(ctx, authpkg.ProviderVK, state)
	if err != nil {
		return AuthResult{}, err
	}
	id, err := s.vk.SignIn(ctx, code, row.CodeVerifier, deviceID, state)
	if err != nil {
		return AuthResult{}, mapProviderErr(err)
	}
	return s.signInExternal(ctx, id, "sign_in_vk", dev)
}

func (s *Service) consumeState(ctx context.Context, provider, state string) (db.OauthState, error) {
	row, err := s.store.ConsumeOAuthState(ctx, db.ConsumeOAuthStateParams{
		StateHash: authpkg.HashOAuthState(state),
		Provider:  provider,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return db.OauthState{}, ErrInvalidOAuthState
	}
	if err != nil {
		return db.OauthState{}, fmt.Errorf("auth: consume oauth state: %w", err)
	}
	return row, nil
}

func mapProviderErr(err error) error {
	if errors.Is(err, authpkg.ErrOAuthRejected) {
		return fmt.Errorf("%w: %v", ErrOAuthFailed, err)
	}
	return fmt.Errorf("%w: %v", ErrOAuthUnavailable, err)
}

func (s *Service) signInExternal(ctx context.Context, id authpkg.ExternalIdentity, action string, dev DeviceMeta) (AuthResult, error) {
	var result AuthResult
	err := s.store.ExecTx(ctx, func(q *db.Queries) error {
		user, err := s.resolveExternalUser(ctx, q, id)
		if err != nil {
			return err
		}
		result, err = s.issueTokens(ctx, q, user, dev)
		if err != nil {
			return err
		}
		s.audit(ctx, q, user.ID, action, dev.IP)
		return nil
	})
	if err != nil {
		return AuthResult{}, err
	}
	return result, nil
}

// resolveExternalUser implements the account resolution order:
//  1. match by provider subject (yandex_sub/vk_sub) → existing account;
//  2. else auto-link by provider-verified email to an existing account;
//  3. else create a fresh account.
func (s *Service) resolveExternalUser(ctx context.Context, q *db.Queries, id authpkg.ExternalIdentity) (db.User, error) {
	if user, err := s.getUserBySub(ctx, q, id.Provider, id.Subject); err == nil {
		return user, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, fmt.Errorf("auth: lookup by sub: %w", err)
	}

	email := normalizeEmail(id.Email)

	if canLinkByEmail(email, id.EmailVerified) {
		user, err := q.GetUserByEmail(ctx, textOrNull(email))
		switch {
		case err == nil:
			return s.linkExisting(ctx, q, id.Provider, user, id.Subject)
		case errors.Is(err, pgx.ErrNoRows):
			// fall through to creation
		default:
			return db.User{}, fmt.Errorf("auth: lookup by email: %w", err)
		}
	}

	storeEmail, verified := emailForStorage(email, id.EmailVerified)
	params := db.CreateUserParams{
		Email:         textOrNull(storeEmail),
		EmailVerified: verified,
		DisplayName:   textOrNull(""),
		Column6:       "", // locale → COALESCE NULLIF default 'ru'
	}
	switch id.Provider {
	case authpkg.ProviderYandex:
		params.YandexSub = textOrNull(id.Subject)
	case authpkg.ProviderVK:
		params.VkSub = textOrNull(id.Subject)
	}
	user, err := q.CreateUser(ctx, params)
	if err != nil {
		return db.User{}, fmt.Errorf("auth: create user: %w", err)
	}
	return user, nil
}

func (s *Service) linkExisting(ctx context.Context, q *db.Queries, provider string, user db.User, sub string) (db.User, error) {
	linked, err := s.linkUserSub(ctx, q, provider, user.ID, sub)
	if err != nil {
		return db.User{}, fmt.Errorf("auth: link sub: %w", err)
	}
	// Both the OAuth provider and a prior registration now vouch for the email.
	if !linked.EmailVerified {
		if err := q.MarkEmailVerified(ctx, linked.ID); err != nil {
			return db.User{}, fmt.Errorf("auth: mark verified: %w", err)
		}
		linked.EmailVerified = true
	}
	return linked, nil
}

func (s *Service) getUserBySub(ctx context.Context, q *db.Queries, provider, sub string) (db.User, error) {
	switch provider {
	case authpkg.ProviderYandex:
		return q.GetUserByYandexSub(ctx, textOrNull(sub))
	case authpkg.ProviderVK:
		return q.GetUserByVKSub(ctx, textOrNull(sub))
	default:
		return db.User{}, fmt.Errorf("auth: unknown provider %q", provider)
	}
}

func (s *Service) linkUserSub(ctx context.Context, q *db.Queries, provider string, id uuid.UUID, sub string) (db.User, error) {
	switch provider {
	case authpkg.ProviderYandex:
		return q.LinkYandexSub(ctx, db.LinkYandexSubParams{ID: id, YandexSub: textOrNull(sub)})
	case authpkg.ProviderVK:
		return q.LinkVKSub(ctx, db.LinkVKSubParams{ID: id, VkSub: textOrNull(sub)})
	default:
		return db.User{}, fmt.Errorf("auth: unknown provider %q", provider)
	}
}

func normalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// canLinkByEmail decides whether an OAuth sign-in may attach to an existing
// account found by email. Only provider-verified addresses qualify (Yandex);
// VK ID gives no confirmation guarantee, so VK never auto-links — preventing
// an attacker with an unconfirmed VK email from taking over an OTP account.
func canLinkByEmail(email string, emailVerified bool) bool {
	return email != "" && emailVerified
}

// emailForStorage decides what email (if any) to persist on a freshly created
// OAuth account. Unverified addresses are dropped: storing one would squat the
// unique users.email slot and block its real owner from OTP registration.
func emailForStorage(email string, emailVerified bool) (store string, verified bool) {
	if email == "" || !emailVerified {
		return "", false
	}
	return email, true
}

func ipText(dev DeviceMeta) pgtype.Text {
	if !dev.IP.IsValid() {
		return pgtype.Text{}
	}
	return textOrNull(dev.IP.String())
}
