package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

const applePrivateRelaySuffix = "@privaterelay.appleid.com"

// SignInApple verifies an Apple id_token (with nonce) and signs the user in,
// creating or linking the account as needed.
func (s *Service) SignInApple(ctx context.Context, idToken, nonce string, dev DeviceMeta) (AuthResult, error) {
	id, err := s.verifier.VerifyApple(ctx, idToken, nonce)
	if err != nil {
		return AuthResult{}, fmt.Errorf("%w: %v", ErrInvalidIDToken, err)
	}
	return s.signInExternal(ctx, id, "sign_in_apple", dev)
}

// SignInGoogle verifies a Google id_token and signs the user in.
func (s *Service) SignInGoogle(ctx context.Context, idToken string, dev DeviceMeta) (AuthResult, error) {
	id, err := s.verifier.VerifyGoogle(ctx, idToken)
	if err != nil {
		return AuthResult{}, fmt.Errorf("%w: %v", ErrInvalidIDToken, err)
	}
	return s.signInExternal(ctx, id, "sign_in_google", dev)
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
//  1. match by provider subject (apple_sub/google_sub) → existing account;
//  2. else auto-link by verified, non-relay email to an existing account;
//  3. else create a fresh account.
func (s *Service) resolveExternalUser(ctx context.Context, q *db.Queries, id authpkg.ExternalIdentity) (db.User, error) {
	if user, err := s.getUserBySub(ctx, q, id.Provider, id.Subject); err == nil {
		return user, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return db.User{}, fmt.Errorf("auth: lookup by sub: %w", err)
	}

	email := normalizeEmail(id.Email)

	if canLinkByEmail(id.Provider, email, id.EmailVerified) {
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

	storeEmail, verified := emailForStorage(id.Provider, email, id.EmailVerified)
	params := db.CreateUserParams{
		Email:         textOrNull(storeEmail),
		EmailVerified: verified,
		DisplayName:   textOrNull(""),
		Column6:       "", // locale → COALESCE NULLIF default 'ru'
	}
	switch id.Provider {
	case "apple":
		params.AppleSub = textOrNull(id.Subject)
	case "google":
		params.GoogleSub = textOrNull(id.Subject)
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
	case "apple":
		return q.GetUserByAppleSub(ctx, textOrNull(sub))
	case "google":
		return q.GetUserByGoogleSub(ctx, textOrNull(sub))
	default:
		return db.User{}, fmt.Errorf("auth: unknown provider %q", provider)
	}
}

func (s *Service) linkUserSub(ctx context.Context, q *db.Queries, provider string, id uuid.UUID, sub string) (db.User, error) {
	switch provider {
	case "apple":
		return q.LinkAppleSub(ctx, db.LinkAppleSubParams{ID: id, AppleSub: textOrNull(sub)})
	case "google":
		return q.LinkGoogleSub(ctx, db.LinkGoogleSubParams{ID: id, GoogleSub: textOrNull(sub)})
	default:
		return db.User{}, fmt.Errorf("auth: unknown provider %q", provider)
	}
}

func normalizeEmail(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func isApplePrivateRelay(email string) bool {
	return strings.HasSuffix(email, applePrivateRelaySuffix)
}

// canLinkByEmail decides whether an OAuth sign-in may attach to an existing
// account found by email. Requires a real (non-relay) address; Google must
// additionally assert email_verified. Apple addresses are provider-verified.
func canLinkByEmail(provider, email string, emailVerified bool) bool {
	if email == "" || isApplePrivateRelay(email) {
		return false
	}
	if provider == "google" && !emailVerified {
		return false
	}
	return true
}

// emailForStorage decides what email (if any) to persist on a freshly created
// OAuth account, and whether to mark it verified. Unverified Google emails are
// dropped to avoid colliding with / impersonating an existing verified account.
func emailForStorage(provider, email string, emailVerified bool) (store string, verified bool) {
	if email == "" {
		return "", false
	}
	if isApplePrivateRelay(email) {
		return email, true // relay addresses are unique and deliverable
	}
	if provider == "google" && !emailVerified {
		return "", false
	}
	return email, true
}
