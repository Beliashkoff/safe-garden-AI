// Package admin implements the admin panel usecases: the single-operator
// account (setup / sign-in / password reset, all pinned to ADMIN_EMAIL),
// catalog management for the fertilizers the model recommends, dashboard
// statistics and the server error feed.
package admin

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/mailer"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

const (
	// codeTTL — one-time code lifetime, mirrors the user OTP (ARCH §8.1).
	codeTTL = 10 * time.Minute
	// maxCodeAttempts — ≤5 verification attempts per issued code.
	maxCodeAttempts = 5
	// maxCodesPerHour — issuance cap per purpose.
	maxCodesPerHour = 3
	// maxFailedLogins / failedLoginWindow — per-IP brute-force guard.
	maxFailedLogins   = 5
	failedLoginWindow = 15 * time.Minute
	// maxFailedLoginsGlobal — account-global cap (IP-independent) that holds even
	// when an attacker rotates spoofed X-Forwarded-For headers to dodge the
	// per-IP counter. Generous enough that the lone operator never trips it in
	// normal use; self-heals after failedLoginWindow.
	maxFailedLoginsGlobal = 20
	// sessionTouchInterval — how stale last_seen_at must be before the sliding
	// TTL is bumped (avoids an UPDATE on every request).
	sessionTouchInterval = 5 * time.Minute
	// absoluteSessionMax — hard ceiling on session age regardless of activity,
	// forcing periodic re-authentication even for a continuously-used session.
	absoluteSessionMax = 30 * 24 * time.Hour
)

// Sentinel errors mapped to HTTP shapes in transport/http/adminapi.
var (
	ErrAlreadyBootstrapped = errors.New("admin: already bootstrapped")
	ErrNotBootstrapped     = errors.New("admin: not bootstrapped")
	ErrEmailNotAllowed     = errors.New("admin: email not allowed")
	ErrInvalidCredentials  = errors.New("admin: invalid credentials")
	ErrInvalidCode         = errors.New("admin: invalid or expired code")
	ErrTooManyAttempts     = errors.New("admin: too many attempts")
	ErrRateLimited         = errors.New("admin: rate limited")
	ErrWeakPassword        = errors.New("admin: password too weak")
	ErrSessionInvalid      = errors.New("admin: session invalid")
	ErrProductNotFound     = errors.New("admin: product not found")
	ErrSlugTaken           = errors.New("admin: slug already exists")
	ErrValidation          = errors.New("admin: validation failed")
)

// objStore is the storage subset the catalog image upload needs (consumer-side
// interface). Satisfied by *objstore.Client and objstore.Disabled.
type objStore interface {
	PutPublic(ctx context.Context, key, contentType string, data []byte) (string, error)
}

// Service implements the admin panel usecases. Like usecase/auth it depends on
// the concrete storage.Store; DB-touching paths are covered by integration
// tests, pure helpers by unit tests.
type Service struct {
	store      *storage.Store
	mailer     mailer.AdminMailer
	objs       objStore
	adminEmail string // normalized (lower-case); the only allowed operator
	sessionTTL time.Duration
	logger     *slog.Logger
	now        func() time.Time
	// dummyHash is a precomputed bcrypt hash compared against on login misses so
	// the work factor is constant regardless of whether the email matched,
	// closing the email-enumeration timing oracle.
	dummyHash []byte
}

// NewService wires the usecase. adminEmail is the pinned operator address
// (ADMIN_EMAIL); the service must not be constructed when it is empty.
func NewService(
	store *storage.Store,
	m mailer.AdminMailer,
	objs objStore,
	adminEmail string,
	sessionTTL time.Duration,
	logger *slog.Logger,
) *Service {
	// Precompute a throwaway bcrypt hash for constant-time login. Cost is the
	// same as a real verify; a failure here is impossible for a constant input,
	// but if it ever did, Login falls back to comparing against nil (still a
	// non-nil-error, uniform failure).
	dummy, err := authpkg.HashPassword("constant-time-dummy-password")
	if err != nil {
		logger.Error("admin: dummy hash init failed", "err", err.Error())
	}
	return &Service{
		store:      store,
		mailer:     m,
		objs:       objs,
		adminEmail: normalizeEmail(adminEmail),
		sessionTTL: sessionTTL,
		logger:     logger,
		now:        time.Now,
		dummyHash:  dummy,
	}
}

// AdminView is the transport-agnostic projection of the operator account.
type AdminView struct {
	ID          uuid.UUID
	Email       string
	CreatedAt   time.Time
	LastLoginAt time.Time // zero when never
}

// SessionView is one active session for the "Сессии" settings list.
type SessionView struct {
	ID         uuid.UUID
	IP         string
	UserAgent  string
	CreatedAt  time.Time
	LastSeenAt time.Time
	Current    bool
}

// SessionResult is returned by every successful sign-in.
type SessionResult struct {
	Token     string
	ExpiresAt time.Time
	Admin     AdminView
}

// DeviceMeta is optional per-request context recorded with sessions and audit.
type DeviceMeta struct {
	UserAgent string
	IP        netip.Addr
}

func toAdminView(a db.AdminUser) AdminView {
	v := AdminView{ID: a.ID, Email: a.Email, CreatedAt: a.CreatedAt.Time}
	if a.LastLoginAt.Valid {
		v.LastLoginAt = a.LastLoginAt.Time
	}
	return v
}

// audit records an admin action. Failures are logged, never propagated.
// details must stay PII-free: catalog names are fine, user data is not.
func (s *Service) audit(ctx context.Context, adminID uuid.UUID, action, entity, entityID string, ip netip.Addr) {
	var aid pgtype.UUID
	if adminID != uuid.Nil {
		aid = pgtype.UUID{Bytes: adminID, Valid: true}
	}
	var ipp *netip.Addr
	if ip.IsValid() {
		ipp = &ip
	}
	if err := s.store.InsertAdminAudit(ctx, db.InsertAdminAuditParams{
		AdminID:  aid,
		Action:   action,
		Entity:   textOrNull(entity),
		EntityID: textOrNull(entityID),
		Ip:       ipp,
	}); err != nil {
		s.logger.ErrorContext(ctx, "admin audit insert failed", "action", action, "err", err.Error())
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validPassword enforces the operator password policy: 12–72 bytes (bcrypt
// cap), at least one letter and one digit.
func validPassword(pw string) bool {
	if len(pw) < 12 || len(pw) > 72 {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range pw {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127:
			hasLetter = true
		}
	}
	return hasLetter && hasDigit
}

func textOrNull(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func timestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}
