package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/mailer"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/ratelimit"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

const (
	// otpTTL — code lifetime (ARCH §8.1).
	otpTTL = 10 * time.Minute
	// maxOTPVerifyAttempts — ≤5 verification attempts per issued code.
	maxOTPVerifyAttempts = 5
)

// Service implements the authentication flows. It depends on the concrete
// storage.Store (which wraps sqlc + ExecTx); the DB-touching logic is covered by
// integration tests, while pure helpers (email/locale/linking) are unit-tested.
type Service struct {
	store      *storage.Store
	issuer     *authpkg.Issuer
	verifier   *authpkg.Verifier
	mailer     mailer.Mailer
	limiter    ratelimit.Limiter
	refreshTTL time.Duration
	logger     *slog.Logger
	now        func() time.Time
}

// NewService wires the usecase. now defaults to time.Now when nil (tests inject
// a fixed clock).
func NewService(
	store *storage.Store,
	issuer *authpkg.Issuer,
	verifier *authpkg.Verifier,
	m mailer.Mailer,
	limiter ratelimit.Limiter,
	refreshTTL time.Duration,
	logger *slog.Logger,
) *Service {
	return &Service{
		store:      store,
		issuer:     issuer,
		verifier:   verifier,
		mailer:     m,
		limiter:    limiter,
		refreshTTL: refreshTTL,
		logger:     logger,
		now:        time.Now,
	}
}

// UserView is the transport-agnostic projection of a user (no pgtype leakage).
type UserView struct {
	ID            uuid.UUID
	Email         string // "" when unset
	EmailVerified bool
	DisplayName   string // "" when unset
	HasApple      bool
	HasGoogle     bool
}

// AuthResult is returned by every successful sign-in / refresh.
type AuthResult struct {
	AccessToken  string
	AccessExp    time.Time
	RefreshToken string
	User         UserView
}

// DeviceMeta is optional per-request context recorded with the session and
// audit log. All fields may be zero.
type DeviceMeta struct {
	UserAgent string
	DeviceID  string
	IP        netip.Addr
}

func toView(u db.User) UserView {
	return UserView{
		ID:            u.ID,
		Email:         u.Email.String,
		EmailVerified: u.EmailVerified,
		DisplayName:   u.DisplayName.String,
		HasApple:      u.AppleSub.Valid,
		HasGoogle:     u.GoogleSub.Valid,
	}
}

// issueTokens mints an access JWT and a rotated refresh token, persisting the
// refresh hash via q (which may be a transaction). The raw refresh value is
// returned to the caller and never stored.
func (s *Service) issueTokens(ctx context.Context, q *db.Queries, user db.User, dev DeviceMeta) (AuthResult, error) {
	access, exp, err := s.issuer.Issue(user.ID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("auth: issue access: %w", err)
	}
	raw, hash, err := authpkg.NewRefreshToken()
	if err != nil {
		return AuthResult{}, fmt.Errorf("auth: new refresh: %w", err)
	}
	if _, err := q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: hash,
		DeviceID:  textOrNull(dev.DeviceID),
		UserAgent: textOrNull(dev.UserAgent),
		ExpiresAt: timestamptz(s.now().Add(s.refreshTTL)),
	}); err != nil {
		return AuthResult{}, fmt.Errorf("auth: persist refresh: %w", err)
	}
	return AuthResult{
		AccessToken:  access,
		AccessExp:    exp,
		RefreshToken: raw,
		User:         toView(user),
	}, nil
}

// audit records a sensitive event. Failures are logged, not propagated — losing
// an audit row must not fail the user-facing action. Details are intentionally
// omitted to avoid persisting PII (CLAUDE.md §3).
func (s *Service) audit(ctx context.Context, q *db.Queries, userID uuid.UUID, action string, ip netip.Addr) {
	var uid pgtype.UUID
	if userID != uuid.Nil {
		uid = pgtype.UUID{Bytes: userID, Valid: true}
	}
	var ipp *netip.Addr
	if ip.IsValid() {
		ipp = &ip
	}
	if err := q.InsertAuditLog(ctx, db.InsertAuditLogParams{
		UserID: uid,
		Action: action,
		Ip:     ipp,
	}); err != nil {
		s.logger.ErrorContext(ctx, "audit log insert failed", "action", action, "err", err.Error())
	}
}

func textOrNull(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// validEmail accepts a normalized address: non-empty, ≤254 chars, parseable,
// and bare (no display name).
func validEmail(email string) bool {
	if email == "" || len(email) > 254 {
		return false
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return addr.Address == email
}

func validOTPFormat(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
