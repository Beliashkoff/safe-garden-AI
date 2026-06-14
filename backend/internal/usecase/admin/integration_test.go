//go:build integration

package admin

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	fertilizeruc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/fertilizer"
)

const adminEmail = "agronomai@yandex.com"

var (
	testStore *storage.Store
	rawPool   *pgxpool.Pool // direct pool for TRUNCATE between tests (Store has no public Exec)
	testCtx   = context.Background()
)

// fakeMailer captures the last code+purpose so the test can complete the flow.
type fakeMailer struct {
	lastCode    string
	lastPurpose string
	lastTo      string
}

func (m *fakeMailer) SendAdminCode(_ context.Context, to, code, purpose string) error {
	m.lastTo, m.lastCode, m.lastPurpose = to, code, purpose
	return nil
}

// fakeObjStore returns a deterministic public URL.
type fakeObjStore struct{ lastKey string }

func (o *fakeObjStore) PutPublic(_ context.Context, key, _ string, _ []byte) (string, error) {
	o.lastKey = key
	return "https://cdn.example/" + key, nil
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
				WithOccurrence(2).WithStartupTimeout(60*time.Second),
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
	s, err := storage.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "storage init: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()
	testStore = s

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "raw pool: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	rawPool = pool

	os.Exit(m.Run())
}

func applyMigrations(dsn string) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("dialect: %w", err)
	}
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "migrations"))
	return goose.Up(conn, dir)
}

func reset(t *testing.T) {
	t.Helper()
	_, err := rawPool.Exec(testCtx,
		"TRUNCATE admin_users, admin_sessions, admin_codes, admin_audit_log, "+
			"admin_error_events, fertilizers RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func newService(m *fakeMailer, o *fakeObjStore) *Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewService(testStore, m, o, adminEmail, time.Hour, logger)
}

var dev = DeviceMeta{UserAgent: "test", IP: netip.MustParseAddr("203.0.113.7")}

// --- auth flow --------------------------------------------------------------

func TestAdminSetupLoginSessionFlow(t *testing.T) {
	t.Cleanup(func() { reset(t) })
	mailer := &fakeMailer{}
	svc := newService(mailer, &fakeObjStore{})

	ok, err := svc.Bootstrapped(testCtx)
	require.NoError(t, err)
	assert.False(t, ok)

	// Setup code only for the pinned email.
	require.ErrorIs(t, svc.RequestSetupCode(testCtx, "intruder@evil.com"), ErrEmailNotAllowed)
	require.NoError(t, svc.RequestSetupCode(testCtx, adminEmail))
	require.Equal(t, "setup", mailer.lastPurpose)
	code := mailer.lastCode

	// Wrong code is rejected.
	_, err = svc.CompleteSetup(testCtx, adminEmail, "000000", "longpassword123", dev)
	require.ErrorIs(t, err, ErrInvalidCode)

	// Weak password is rejected even with a valid code (code stays usable).
	_, err = svc.CompleteSetup(testCtx, adminEmail, code, "short", dev)
	require.ErrorIs(t, err, ErrWeakPassword)

	res, err := svc.CompleteSetup(testCtx, adminEmail, code, "longpassword123", dev)
	require.NoError(t, err)
	require.NotEmpty(t, res.Token)
	assert.Equal(t, adminEmail, res.Admin.Email)

	// Bootstrapped now; setup is closed.
	ok, err = svc.Bootstrapped(testCtx)
	require.NoError(t, err)
	assert.True(t, ok)
	require.ErrorIs(t, svc.RequestSetupCode(testCtx, adminEmail), ErrAlreadyBootstrapped)

	// Session validates, then logout invalidates it.
	sess, err := svc.ValidateSession(testCtx, res.Token)
	require.NoError(t, err)
	assert.Equal(t, res.Admin.ID, sess.AdminID)
	require.NoError(t, svc.Logout(testCtx, res.Token))
	_, err = svc.ValidateSession(testCtx, res.Token)
	require.ErrorIs(t, err, ErrSessionInvalid)
}

func TestAdminLoginRejectsWrongCredentialsAndForeignEmail(t *testing.T) {
	t.Cleanup(func() { reset(t) })
	svc := newService(&fakeMailer{}, &fakeObjStore{})
	bootstrap(t, svc)

	_, err := svc.Login(testCtx, adminEmail, "wrong-password-xx", dev)
	require.ErrorIs(t, err, ErrInvalidCredentials)

	// Even with a (hypothetically) correct password, a non-pinned email cannot log in.
	_, err = svc.Login(testCtx, "intruder@evil.com", "longpassword123", dev)
	require.ErrorIs(t, err, ErrInvalidCredentials)

	res, err := svc.Login(testCtx, adminEmail, "longpassword123", dev)
	require.NoError(t, err)
	require.NotEmpty(t, res.Token)
}

func TestAdminLoginBruteForceGuard(t *testing.T) {
	t.Cleanup(func() { reset(t) })
	svc := newService(&fakeMailer{}, &fakeObjStore{})
	bootstrap(t, svc)

	for i := 0; i < maxFailedLogins; i++ {
		_, err := svc.Login(testCtx, adminEmail, "definitely-wrong", dev)
		require.ErrorIs(t, err, ErrInvalidCredentials)
	}
	// Next attempt from the same IP is rate-limited even with the right password.
	_, err := svc.Login(testCtx, adminEmail, "longpassword123", dev)
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestAdminPasswordResetRevokesSessions(t *testing.T) {
	t.Cleanup(func() { reset(t) })
	mailer := &fakeMailer{}
	svc := newService(mailer, &fakeObjStore{})
	bootstrap(t, svc)

	login, err := svc.Login(testCtx, adminEmail, "longpassword123", dev)
	require.NoError(t, err)

	require.NoError(t, svc.RequestResetCode(testCtx, adminEmail))
	require.Equal(t, "reset", mailer.lastPurpose)
	require.NoError(t, svc.CompleteReset(testCtx, adminEmail, mailer.lastCode, "brandnewpass456"))

	// Old session revoked; old password no longer works; new one does.
	_, err = svc.ValidateSession(testCtx, login.Token)
	require.ErrorIs(t, err, ErrSessionInvalid)
	_, err = svc.Login(testCtx, adminEmail, "longpassword123", dev)
	require.ErrorIs(t, err, ErrInvalidCredentials)
	_, err = svc.Login(testCtx, adminEmail, "brandnewpass456", dev)
	require.NoError(t, err)
}

// --- catalog flow -----------------------------------------------------------

func TestAdminCatalogCRUDFeedsRecommendation(t *testing.T) {
	t.Cleanup(func() { reset(t) })
	svc := newService(&fakeMailer{}, &fakeObjStore{})
	adminID := bootstrap(t, svc)

	price := int32(499)
	created, err := svc.CreateProduct(testCtx, adminID, ProductInput{
		Name:      "Зелёный рост",
		ShortDesc: "Азотная подкормка для рассады",
		Category:  "минеральные",
		Problems:  []string{"nitrogen_deficiency", "leaf_yellowing"},
		Plants:    []string{"томат"},
		Priority:  10,
		Active:    true,
		PriceRub:  &price,
	}, dev)
	require.NoError(t, err)
	assert.Equal(t, "zelenyy-rost", created.Slug)

	// The recommendation usecase (worker callback) now finds it by problem key.
	rec := fertilizeruc.NewService(testStore)
	products, err := rec.Recommend(testCtx, llm.FertilizerToolArgs{Problem: "nitrogen_deficiency", Plant: "томат"})
	require.NoError(t, err)
	require.Len(t, products, 1)
	assert.Equal(t, "Зелёный рост", products[0].Name)
	require.NotNil(t, products[0].PriceRub)
	assert.Equal(t, int32(499), *products[0].PriceRub)

	// Deactivate → no longer recommended.
	_, err = svc.UpdateProduct(testCtx, adminID, created.ID, ProductInput{
		Name:      created.Name,
		ShortDesc: created.ShortDesc,
		Category:  created.Category,
		Problems:  created.Problems,
		Plants:    created.Plants,
		Active:    false,
	}, dev)
	require.NoError(t, err)
	products, err = rec.Recommend(testCtx, llm.FertilizerToolArgs{Problem: "nitrogen_deficiency"})
	require.NoError(t, err)
	assert.Empty(t, products)

	// Duplicate slug is rejected.
	_, err = svc.CreateProduct(testCtx, adminID, ProductInput{
		Slug: "zelenyy-rost", Name: "Другой", ShortDesc: "x", Category: "y",
		Problems: []string{"wilting"}, Active: true,
	}, dev)
	require.ErrorIs(t, err, ErrSlugTaken)

	// Delete.
	require.NoError(t, svc.DeleteProduct(testCtx, adminID, created.ID, dev))
	list, err := svc.ListProducts(testCtx)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestAdminImageUploadAndOverview(t *testing.T) {
	t.Cleanup(func() { reset(t) })
	objs := &fakeObjStore{}
	svc := newService(&fakeMailer{}, objs)
	adminID := bootstrap(t, svc)

	pngSig := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	url, err := svc.UploadProductImage(testCtx, adminID, "image/png", pngSig, dev)
	require.NoError(t, err)
	assert.Contains(t, url, "catalog/")

	// Declared type not whitelisted.
	_, err = svc.UploadProductImage(testCtx, adminID, "application/pdf", []byte{1}, dev)
	require.ErrorIs(t, err, ErrValidation)

	// Declared png but bytes are not an image → content sniff rejects it.
	_, err = svc.UploadProductImage(testCtx, adminID, "image/png", []byte("<html>nope</html>"), dev)
	require.ErrorIs(t, err, ErrValidation)

	ov, err := svc.GetOverview(testCtx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), ov.CatalogTotal)
}

// bootstrap creates the admin account with password "longpassword123" and
// returns its id. It clears the audit log so failed-login counting in later
// tests starts isolated.
func bootstrap(t *testing.T, svc *Service) uuid.UUID {
	t.Helper()
	m := svc.mailer.(*fakeMailer)
	require.NoError(t, svc.RequestSetupCode(testCtx, adminEmail))
	res, err := svc.CompleteSetup(testCtx, adminEmail, m.lastCode, "longpassword123", dev)
	require.NoError(t, err)
	_, err = rawPool.Exec(testCtx, "DELETE FROM admin_audit_log")
	require.NoError(t, err)
	return res.Admin.ID
}
