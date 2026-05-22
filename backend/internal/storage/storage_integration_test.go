//go:build integration

package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage/db"
)

var (
	testStore *Store
	testCtx   = context.Background()
)

// migrationsDir resolves backend/migrations from this test file's location so
// the resolver does not depend on the current working directory.
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// thisFile: backend/internal/storage/storage_integration_test.go
	// migrations: backend/migrations
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations"))
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "testcontainers start: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = pg.Terminate(ctx) }()

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "conn string: %v\n", err)
		os.Exit(1)
	}

	if err := applyMigrations(dsn); err != nil {
		fmt.Fprintf(os.Stderr, "migrations: %v\n", err)
		os.Exit(1)
	}

	s, err := New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "storage init: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()
	testStore = s

	os.Exit(m.Run())
}

func applyMigrations(dsn string) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer conn.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("dialect: %w", err)
	}
	dir := ""
	// applyMigrations is called from TestMain so runtime.Caller works.
	_, thisFile, _, _ := runtime.Caller(0)
	dir = filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations"))
	return goose.Up(conn, dir)
}

// truncateAll resets all stage-1.1 tables between tests. Using TRUNCATE with
// RESTART IDENTITY + CASCADE keeps audit_log's BIGSERIAL deterministic.
func truncateAll(t *testing.T) {
	t.Helper()
	_, err := testStore.pool.Exec(testCtx,
		"TRUNCATE users, refresh_tokens, email_codes, audit_log RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func textPtr(s string) pgtype.Text { return pgtype.Text{String: s, Valid: true} }

func tsAt(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

// ----- users ----------------------------------------------------------------

func TestCreateUser_AppleSubUnique(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries

	u1, err := q.CreateUser(testCtx, db.CreateUserParams{
		Email:         textPtr("alice@example.com"),
		EmailVerified: false,
		AppleSub:      textPtr("apple-sub-1"),
		GoogleSub:     pgtype.Text{},
		DisplayName:   pgtype.Text{},
		Column6:       "",
	})
	require.NoError(t, err)
	assert.Equal(t, "ru", u1.Locale)

	_, err = q.CreateUser(testCtx, db.CreateUserParams{
		Email:         textPtr("alice2@example.com"),
		EmailVerified: false,
		AppleSub:      textPtr("apple-sub-1"), // duplicate
		GoogleSub:     pgtype.Text{},
		DisplayName:   pgtype.Text{},
		Column6:       "",
	})
	require.Error(t, err)
}

func TestCreateUser_EmailCaseInsensitive(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries

	_, err := q.CreateUser(testCtx, db.CreateUserParams{
		Email:         textPtr("Foo@Example.com"),
		EmailVerified: true,
		Column6:       "",
	})
	require.NoError(t, err)

	u, err := q.GetUserByEmail(testCtx, textPtr("foo@example.com"))
	require.NoError(t, err)
	assert.True(t, u.EmailVerified)
}

func TestGetUserByID_FiltersDeleted(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries

	u, err := q.CreateUser(testCtx, db.CreateUserParams{
		Email:   textPtr("del@example.com"),
		Column6: "",
	})
	require.NoError(t, err)

	require.NoError(t, q.SoftDeleteUser(testCtx, u.ID))

	_, err = q.GetUserByID(testCtx, u.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, sql.ErrNoRows) || err.Error() == "no rows in result set")
}

func TestSoftDeleteUser_FreesIdentifiers(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries

	u, err := q.CreateUser(testCtx, db.CreateUserParams{
		Email:     textPtr("reuse@example.com"),
		AppleSub:  textPtr("apple-x"),
		GoogleSub: textPtr("google-y"),
		Column6:   "",
	})
	require.NoError(t, err)
	require.NoError(t, q.SoftDeleteUser(testCtx, u.ID))

	// Same identifiers can be used by a new user after delete.
	_, err = q.CreateUser(testCtx, db.CreateUserParams{
		Email:     textPtr("reuse@example.com"),
		AppleSub:  textPtr("apple-x"),
		GoogleSub: textPtr("google-y"),
		Column6:   "",
	})
	require.NoError(t, err)
}

// ----- refresh tokens -------------------------------------------------------

func mkUser(t *testing.T) db.User {
	t.Helper()
	u, err := testStore.Queries.CreateUser(testCtx, db.CreateUserParams{
		Email:   textPtr(fmt.Sprintf("u-%d@example.com", time.Now().UnixNano())),
		Column6: "",
	})
	require.NoError(t, err)
	return u
}

func TestRefreshToken_UniqueHash(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries
	u := mkUser(t)

	hash := []byte("fake-sha256-hash-32bytes------xx")
	_, err := q.CreateRefreshToken(testCtx, db.CreateRefreshTokenParams{
		UserID:    u.ID,
		TokenHash: hash,
		DeviceID:  pgtype.Text{},
		UserAgent: pgtype.Text{},
		ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)

	_, err = q.CreateRefreshToken(testCtx, db.CreateRefreshTokenParams{
		UserID:    u.ID,
		TokenHash: hash, // duplicate
		ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.Error(t, err)
}

func TestRevokeAllUserRefreshTokens_OnlyActive(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries
	u := mkUser(t)

	active, err := q.CreateRefreshToken(testCtx, db.CreateRefreshTokenParams{
		UserID: u.ID, TokenHash: []byte("hash-active"), ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)
	old, err := q.CreateRefreshToken(testCtx, db.CreateRefreshTokenParams{
		UserID: u.ID, TokenHash: []byte("hash-old"), ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)
	// Pre-revoke the "old" one.
	require.NoError(t, q.RevokeRefreshToken(testCtx, old.ID))
	pre, err := q.GetRefreshTokenByHash(testCtx, old.TokenHash)
	require.NoError(t, err)
	preRevokedAt := pre.RevokedAt

	require.NoError(t, q.RevokeAllUserRefreshTokens(testCtx, u.ID))

	// Active row is now revoked.
	got, err := q.GetRefreshTokenByHash(testCtx, active.TokenHash)
	require.NoError(t, err)
	assert.True(t, got.RevokedAt.Valid)

	// Pre-revoked row's revoked_at is untouched (revoked_at IS NULL guard).
	gotOld, err := q.GetRefreshTokenByHash(testCtx, old.TokenHash)
	require.NoError(t, err)
	assert.Equal(t, preRevokedAt.Time.Unix(), gotOld.RevokedAt.Time.Unix())
}

func TestRefreshTokenRotation_InTransaction(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	u := mkUser(t)

	oldHash := []byte("rotate-old")
	old, err := testStore.Queries.CreateRefreshToken(testCtx, db.CreateRefreshTokenParams{
		UserID: u.ID, TokenHash: oldHash, ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)

	// Successful rotation: revoke old + create new in a single transaction.
	err = testStore.ExecTx(testCtx, func(q *db.Queries) error {
		if err := q.RevokeRefreshToken(testCtx, old.ID); err != nil {
			return err
		}
		_, err := q.CreateRefreshToken(testCtx, db.CreateRefreshTokenParams{
			UserID: u.ID, TokenHash: []byte("rotate-new"), ExpiresAt: tsAt(time.Now().Add(time.Hour)),
		})
		return err
	})
	require.NoError(t, err)

	got, err := testStore.Queries.GetRefreshTokenByHash(testCtx, oldHash)
	require.NoError(t, err)
	assert.True(t, got.RevokedAt.Valid)

	// Failure path: rotation that errors midway must roll back both statements.
	failHash := []byte("rotate-fail-old")
	failOld, err := testStore.Queries.CreateRefreshToken(testCtx, db.CreateRefreshTokenParams{
		UserID: u.ID, TokenHash: failHash, ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)

	wantErr := errors.New("simulated failure")
	err = testStore.ExecTx(testCtx, func(q *db.Queries) error {
		if err := q.RevokeRefreshToken(testCtx, failOld.ID); err != nil {
			return err
		}
		return wantErr
	})
	require.ErrorIs(t, err, wantErr)

	stillActive, err := testStore.Queries.GetRefreshTokenByHash(testCtx, failHash)
	require.NoError(t, err)
	assert.False(t, stillActive.RevokedAt.Valid, "revoke must have been rolled back")
}

// ----- email codes ----------------------------------------------------------

func TestEmailCode_GetActive_OrderingAndExpiry(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries
	email := "otp@example.com"

	older, err := q.CreateEmailCode(testCtx, db.CreateEmailCodeParams{
		Email:     email,
		CodeHash:  []byte("hash-older"),
		ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	newer, err := q.CreateEmailCode(testCtx, db.CreateEmailCodeParams{
		Email:     email,
		CodeHash:  []byte("hash-newer"),
		ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)

	active, err := q.GetActiveEmailCode(testCtx, email)
	require.NoError(t, err)
	assert.Equal(t, newer.ID, active.ID)
	_ = older

	// Mark newer used → older should now be the active result? Older still
	// has expires_at > now and used_at is NULL → yes, it surfaces.
	require.NoError(t, q.MarkEmailCodeUsed(testCtx, newer.ID))
	active, err = q.GetActiveEmailCode(testCtx, email)
	require.NoError(t, err)
	assert.Equal(t, older.ID, active.ID)

	// Expire older → no active code.
	_, err = testStore.pool.Exec(testCtx,
		"UPDATE email_codes SET expires_at = NOW() - INTERVAL '1 minute' WHERE id = $1", older.ID)
	require.NoError(t, err)
	_, err = q.GetActiveEmailCode(testCtx, email)
	require.Error(t, err)
}

func TestEmailCode_IncrementAttempts(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries

	code, err := q.CreateEmailCode(testCtx, db.CreateEmailCodeParams{
		Email:     "attempts@example.com",
		CodeHash:  []byte("h"),
		ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)
	assert.EqualValues(t, 0, code.Attempts)

	a1, err := q.IncrementEmailCodeAttempts(testCtx, code.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, a1)

	a2, err := q.IncrementEmailCodeAttempts(testCtx, code.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 2, a2)
}

func TestCountRecentEmailCodes_OneHourWindow(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries
	email := "window@example.com"

	// Two recent.
	for i := 0; i < 2; i++ {
		_, err := q.CreateEmailCode(testCtx, db.CreateEmailCodeParams{
			Email: email, CodeHash: []byte(fmt.Sprintf("h-%d", i)),
			ExpiresAt: tsAt(time.Now().Add(time.Hour)),
		})
		require.NoError(t, err)
	}
	// One old (manually backdated).
	old, err := q.CreateEmailCode(testCtx, db.CreateEmailCodeParams{
		Email: email, CodeHash: []byte("h-old"),
		ExpiresAt: tsAt(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)
	_, err = testStore.pool.Exec(testCtx,
		"UPDATE email_codes SET created_at = NOW() - INTERVAL '2 hours' WHERE id = $1", old.ID)
	require.NoError(t, err)

	n, err := q.CountRecentEmailCodes(testCtx, email)
	require.NoError(t, err)
	assert.EqualValues(t, 2, n)
}

// ----- audit log ------------------------------------------------------------

func TestAuditLog_InsertWithAndWithoutUser(t *testing.T) {
	t.Cleanup(func() { truncateAll(t) })
	q := testStore.Queries
	u := mkUser(t)

	require.NoError(t, q.InsertAuditLog(testCtx, db.InsertAuditLogParams{
		UserID:  pgtype.UUID{Bytes: u.ID, Valid: true},
		Action:  "user_event",
		Details: []byte(`{"foo":"bar"}`),
	}))

	// System-level event without a user.
	require.NoError(t, q.InsertAuditLog(testCtx, db.InsertAuditLogParams{
		UserID:  pgtype.UUID{},
		Action:  "system_event",
		Details: nil,
	}))

	rows, err := q.ListAuditByUser(testCtx, db.ListAuditByUserParams{
		UserID: pgtype.UUID{Bytes: u.ID, Valid: true},
		Limit:  10,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "user_event", rows[0].Action)
}
