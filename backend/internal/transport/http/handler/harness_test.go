//go:build integration

package handler_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	authpkg "github.com/Beliashkoff/safe-garden-AI/backend/internal/auth"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/imageconv"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/ratelimit"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/storage"
	httptransport "github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/transport/http/handler"
	authuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/auth"
	chatuc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/chat"
	uploaduc "github.com/Beliashkoff/safe-garden-AI/backend/internal/usecase/upload"
)

const (
	testAppleBundle  = "com.example.app"
	testGoogleClient = "ios.apps.googleusercontent.com"
)

var (
	testStore  *storage.Store
	adminDB    *sql.DB
	testCtx    = context.Background()
	testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
)

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

	adminDB, err = sql.Open("pgx", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "admin db: %v\n", err)
		os.Exit(1)
	}
	defer adminDB.Close()

	testStore, err = storage.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "storage init: %v\n", err)
		os.Exit(1)
	}
	defer testStore.Close()

	os.Exit(m.Run())
}

func applyMigrations(dsn string) error {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	_, thisFile, _, _ := runtime.Caller(0)
	// thisFile: backend/internal/transport/http/handler/harness_test.go
	dir := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "migrations"))
	return goose.Up(conn, dir)
}

func truncateAll(t *testing.T) {
	t.Helper()
	_, err := adminDB.Exec("TRUNCATE users, refresh_tokens, email_codes, audit_log, " +
		"conversations, messages, message_blocks, uploads, fertilizers, usage_log " +
		"RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

// recordingMailer captures the most recent OTP so tests can complete the verify
// step (the stored code is bcrypt-hashed and unreadable).
type recordingMailer struct {
	mu        sync.Mutex
	lastEmail string
	lastCode  string
	sends     int
}

func (m *recordingMailer) SendOTP(_ context.Context, to, code, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastEmail, m.lastCode = to, code
	m.sends++
	return nil
}

func (m *recordingMailer) code() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastCode
}

// chatMsgLimiter mirrors the chat usecase's consumer-side rate-limit interface.
type chatMsgLimiter interface {
	AllowMessage(ctx context.Context, userID uuid.UUID) (bool, error)
}

// fakeObjStore is an in-memory object store for chat/upload tests. PresignPut
// returns a deterministic URL (the real PUT to storage is MinIO's job, not
// ours); Get serves objects seeded via put and records reads.
type fakeObjStore struct {
	mu              sync.Mutex
	objects         map[string]fakeObject
	gets            []string
	deletedPrefixes []string
}

type fakeObject struct {
	data        []byte
	contentType string
}

func newFakeObjStore() *fakeObjStore {
	return &fakeObjStore{objects: make(map[string]fakeObject)}
}

func (f *fakeObjStore) PresignPut(_ context.Context, key, contentType string, _ time.Duration) (string, map[string]string, error) {
	return "http://fake-storage.local/" + key, map[string]string{"Content-Type": contentType}, nil
}

func (f *fakeObjStore) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "http://fake-storage.local/" + key + "?get=1", nil
}

func (f *fakeObjStore) Get(_ context.Context, key string) ([]byte, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets = append(f.gets, key)
	o, ok := f.objects[key]
	if !ok {
		return nil, "", fmt.Errorf("fake objstore: not found: %s", key)
	}
	return o.data, o.contentType, nil
}

func (f *fakeObjStore) put(key string, data []byte, contentType string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = fakeObject{data: data, contentType: contentType}
}

func (f *fakeObjStore) DeletePrefix(_ context.Context, prefix string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deletedPrefixes = append(f.deletedPrefixes, prefix)
	n := 0
	for k := range f.objects {
		if strings.HasPrefix(k, prefix) {
			delete(f.objects, k)
			n++
		}
	}
	return n, nil
}

func (f *fakeObjStore) DeleteKeys(_ context.Context, keys []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, k := range keys {
		delete(f.objects, k)
	}
	return nil
}

func (f *fakeObjStore) prefixDeleted(prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, p := range f.deletedPrefixes {
		if p == prefix {
			return true
		}
	}
	return false
}

func (f *fakeObjStore) getCount(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, g := range f.gets {
		if g == key {
			n++
		}
	}
	return n
}

// harness is a full HTTP stack wired against the shared test Postgres, a fresh
// fake IdP, a recording mailer, and a mock LLM client (mutable by chat tests).
type harness struct {
	srv    *httptest.Server
	mailer *recordingMailer
	idp    *fakeIDP
	issuer *authpkg.Issuer
	mock   *llm.MockClient
	objs   *fakeObjStore
}

type harnessConfig struct {
	limiter chatMsgLimiter
}

type harnessOpt func(*harnessConfig)

// withMessageLimiter overrides the (default no-op) chat rate limiter.
func withMessageLimiter(l chatMsgLimiter) harnessOpt {
	return func(c *harnessConfig) { c.limiter = l }
}

func newHarness(t *testing.T, opts ...harnessOpt) *harness {
	t.Helper()
	truncateAll(t)

	cfg := harnessConfig{limiter: ratelimit.NewNoopMessage()}
	for _, o := range opts {
		o(&cfg)
	}

	idp := newFakeIDP(t)
	t.Cleanup(idp.Close)

	verifier, err := authpkg.NewVerifier(testCtx, authpkg.VerifierConfig{
		AppleBundleID:        testAppleBundle,
		AppleIssuerOverride:  idp.issuer,
		GoogleClientIOS:      testGoogleClient,
		GoogleIssuerOverride: idp.issuer,
	})
	require.NoError(t, err)

	issuer := newTestIssuer(t)
	rec := &recordingMailer{}
	authService := authuc.NewService(testStore, issuer, verifier, rec,
		ratelimit.NewDB(testStore), 720*time.Hour, testLogger)

	mock := llm.NewMockClient()
	objs := newFakeObjStore()
	uploadService := uploaduc.NewService(testStore, objs)
	chatService := chatuc.NewService(
		testStore, mock, cfg.limiter, objs, imageconv.New(),
		"test-pepper", llm.DefaultModel, testLogger,
	)

	root := chi.NewRouter()
	root.Use(chimw.RequestID)
	root.Use(chimw.RealIP)
	root.Use(chimw.Recoverer)
	root.Mount("/v1", httptransport.NewRouter(httptransport.Deps{
		Handler:     handler.New(authService, chatService, uploadService),
		TokenParser: issuer,
		DocsEnabled: true,
	}))

	srv := httptest.NewServer(root)
	t.Cleanup(srv.Close)

	return &harness{srv: srv, mailer: rec, idp: idp, issuer: issuer, mock: mock, objs: objs}
}

func newTestIssuer(t *testing.T) *authpkg.Issuer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	der := x509.MarshalPKCS1PrivateKey(key)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})
	path := filepath.Join(t.TempDir(), "jwt.pem")
	require.NoError(t, os.WriteFile(path, pemBytes, 0o600))
	iss, err := authpkg.NewIssuer(authpkg.IssuerConfig{
		PrivateKeyPath: path,
		KID:            "test",
		AccessTTL:      15 * time.Minute,
	})
	require.NoError(t, err)
	return iss
}

// --- HTTP helpers ---

func (h *harness) do(t *testing.T, method, path string, body any, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		var buf bytes.Buffer
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
		rdr = &buf
	}
	req, err := http.NewRequest(method, h.srv.URL+path, rdr)
	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := h.srv.Client().Do(req)
	require.NoError(t, err)
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	return resp, data
}

func (h *harness) postJSON(t *testing.T, path string, body any) (*http.Response, []byte) {
	return h.do(t, http.MethodPost, path, body, nil)
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// --- response shapes ---

type signInResp struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Providers     struct {
			Apple  bool `json:"apple"`
			Google bool `json:"google"`
			Email  bool `json:"email"`
		} `json:"providers"`
	} `json:"user"`
}

type errorResp struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	RequestID string `json:"request_id"`
}

// --- fake IdP (mirrors internal/auth oidc_test.go) ---

type fakeIDP struct {
	srv    *httptest.Server
	key    *rsa.PrivateKey
	kid    string
	issuer string
}

func newFakeIDP(t *testing.T) *fakeIDP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	idp := &fakeIDP{key: key, kid: "idp-key"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"issuer":"%s","jwks_uri":"%s/jwks","authorization_endpoint":"%s/auth","token_endpoint":"%s/token","response_types_supported":["id_token"],"subject_types_supported":["public"],"id_token_signing_alg_values_supported":["RS256"]}`,
			idp.issuer, idp.issuer, idp.issuer, idp.issuer)
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
		e := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())
		fmt.Fprintf(w, `{"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":"%s","n":"%s","e":"%s"}]}`, idp.kid, n, e)
	})
	idp.srv = httptest.NewServer(mux)
	idp.issuer = idp.srv.URL
	return idp
}

func (idp *fakeIDP) Close() { idp.srv.Close() }

func (idp *fakeIDP) appleToken(t *testing.T, sub, email, nonce string, emailVerified bool) string {
	return idp.sign(t, jwt.MapClaims{
		"iss":            idp.issuer,
		"aud":            testAppleBundle,
		"sub":            sub,
		"iat":            time.Now().Add(-time.Minute).Unix(),
		"exp":            time.Now().Add(10 * time.Minute).Unix(),
		"email":          email,
		"email_verified": emailVerified,
		"nonce":          nonce,
	})
}

func (idp *fakeIDP) googleToken(t *testing.T, sub, email string, emailVerified bool) string {
	return idp.sign(t, jwt.MapClaims{
		"iss":            idp.issuer,
		"aud":            testGoogleClient,
		"sub":            sub,
		"iat":            time.Now().Add(-time.Minute).Unix(),
		"exp":            time.Now().Add(10 * time.Minute).Unix(),
		"email":          email,
		"email_verified": emailVerified,
	})
}

func (idp *fakeIDP) sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = idp.kid
	out, err := tok.SignedString(idp.key)
	require.NoError(t, err)
	return out
}
