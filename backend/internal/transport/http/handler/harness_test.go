//go:build integration

package handler_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
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

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/audio"
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
	testYandexClientID = "ya-client-id"
	testYandexSecret   = "ya-client-secret"
	testYandexRedirect = "safegarden://auth/yandex"
	testVKClientID     = "53000000"
	testVKRedirect     = "vk53000000://vk.ru"
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
		"conversations, messages, message_blocks, message_feedback, uploads, fertilizers, " +
		"usage_log, oauth_states RESTART IDENTITY CASCADE")
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

func (f *fakeObjStore) GetLimited(ctx context.Context, key string, _ int64) ([]byte, string, error) {
	return f.Get(ctx, key)
}

// fakeConverter stands in for the ffmpeg converter so integration tests need no
// ffmpeg binary. It returns fixed OggOpus bytes + duration, or err when set.
type fakeConverter struct {
	ogg        []byte
	durationMs int64
	err        error
}

func (f *fakeConverter) ToOggOpus(context.Context, []byte, string) ([]byte, int64, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.ogg, f.durationMs, nil
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

// harness is a full HTTP stack wired against the shared test Postgres, fresh
// fake OAuth providers, a recording mailer, and a mock LLM client (mutable by
// chat tests).
type harness struct {
	srv    *httptest.Server
	mailer *recordingMailer
	yandex *fakeYandex
	vk     *fakeVK
	issuer *authpkg.Issuer
	mock   *llm.MockClient
	objs   *fakeObjStore
	stt    *audio.MockTranscriber
	conv   *fakeConverter
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

	fakeYa := newFakeYandex(t)
	t.Cleanup(fakeYa.Close)
	fakeVKSrv := newFakeVK(t)
	t.Cleanup(fakeVKSrv.Close)

	yandexClient := authpkg.NewYandex(authpkg.YandexConfig{
		ClientID:     testYandexClientID,
		ClientSecret: testYandexSecret,
		RedirectURI:  testYandexRedirect,
		OAuthBaseURL: fakeYa.srv.URL,
		LoginBaseURL: fakeYa.srv.URL,
	})
	vkClient := authpkg.NewVK(authpkg.VKConfig{
		ClientID:    testVKClientID,
		RedirectURI: testVKRedirect,
		BaseURL:     fakeVKSrv.srv.URL,
	})

	issuer := newTestIssuer(t)
	rec := &recordingMailer{}
	authService := authuc.NewService(testStore, issuer, yandexClient, vkClient, rec,
		ratelimit.NewDB(testStore), 720*time.Hour, testLogger)

	mock := llm.NewMockClient()
	objs := newFakeObjStore()
	stt := audio.NewMockTranscriber()
	conv := &fakeConverter{ogg: []byte("ogg-bytes"), durationMs: 4200}
	uploadService := uploaduc.NewService(testStore, objs)
	chatService := chatuc.NewService(
		testStore, mock, cfg.limiter, objs, imageconv.New(),
		stt, conv, "test-pepper", llm.DefaultModel, "ru-RU", testLogger,
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

	return &harness{srv: srv, mailer: rec, yandex: fakeYa, vk: fakeVKSrv, issuer: issuer, mock: mock, objs: objs, stt: stt, conv: conv}
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
			Yandex bool `json:"yandex"`
			VK     bool `json:"vk"`
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

// --- fake OAuth providers ---

type providerIdentity struct {
	sub   string
	email string
}

// fakeYandex emulates oauth.yandex.ru/token + login.yandex.ru/info?format=jwt.
// Codes are registered per test via addCode; the /info response is an HS256
// JWT signed with the client_secret, exactly like the real endpoint.
type fakeYandex struct {
	srv *httptest.Server

	mu     sync.Mutex
	codes  map[string]providerIdentity // auth code → identity
	tokens map[string]providerIdentity // access token → identity
	n      int
}

func newFakeYandex(t *testing.T) *fakeYandex {
	t.Helper()
	f := &fakeYandex{
		codes:  map[string]providerIdentity{},
		tokens: map[string]providerIdentity{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		f.mu.Lock()
		defer f.mu.Unlock()
		id, ok := f.codes[r.PostFormValue("code")]
		if r.PostFormValue("grant_type") != "authorization_code" ||
			r.PostFormValue("client_id") != testYandexClientID ||
			r.PostFormValue("client_secret") != testYandexSecret ||
			r.PostFormValue("code_verifier") == "" || !ok {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"invalid_grant"}`)
			return
		}
		delete(f.codes, r.PostFormValue("code"))
		f.n++
		tok := fmt.Sprintf("ya-access-%d", f.n)
		f.tokens[tok] = id
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"token_type":"bearer","access_token":"%s","expires_in":31536000}`, tok)
	})
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		id, ok := f.tokens[strings.TrimPrefix(r.Header.Get("Authorization"), "OAuth ")]
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		claims := jwt.MapClaims{
			"iss":   "login.yandex.ru",
			"iat":   time.Now().Add(-time.Minute).Unix(),
			"exp":   time.Now().Add(10 * time.Minute).Unix(),
			"uid":   id.sub,
			"login": "tester",
		}
		if id.email != "" {
			claims["email"] = id.email
		}
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
			SignedString([]byte(testYandexSecret))
		require.NoError(t, err)
		fmt.Fprint(w, signed)
	})
	f.srv = httptest.NewServer(mux)
	return f
}

func (f *fakeYandex) Close() { f.srv.Close() }

// addCode registers a one-time auth code for the given identity.
func (f *fakeYandex) addCode(sub, email string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	code := fmt.Sprintf("ya-code-%d", f.n)
	f.codes[code] = providerIdentity{sub: sub, email: email}
	return code
}

// fakeVK emulates id.vk.ru/oauth2/auth + /oauth2/user_info. Codes carry the
// device_id VK would issue next to them; the exchange echoes the request state.
type fakeVK struct {
	srv *httptest.Server

	mu     sync.Mutex
	codes  map[string]vkCode
	tokens map[string]providerIdentity
	n      int
}

type vkCode struct {
	id       providerIdentity
	deviceID string
}

const testVKDeviceID = "vk-device-1"

func newFakeVK(t *testing.T) *fakeVK {
	t.Helper()
	f := &fakeVK{
		codes:  map[string]vkCode{},
		tokens: map[string]providerIdentity{},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/auth", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		f.mu.Lock()
		defer f.mu.Unlock()
		c, ok := f.codes[r.PostFormValue("code")]
		if r.PostFormValue("grant_type") != "authorization_code" ||
			r.PostFormValue("client_id") != testVKClientID ||
			r.PostFormValue("redirect_uri") != testVKRedirect ||
			r.PostFormValue("code_verifier") == "" ||
			!ok || r.PostFormValue("device_id") != c.deviceID {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":"invalid_request"}`)
			return
		}
		delete(f.codes, r.PostFormValue("code"))
		f.n++
		tok := fmt.Sprintf("vk-access-%d", f.n)
		f.tokens[tok] = c.id
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"access_token":"%s","refresh_token":"r","id_token":"i","token_type":"Bearer","expires_in":3600,"user_id":%s,"state":"%s","scope":"vkid.personal_info email"}`,
			tok, c.id.sub, r.PostFormValue("state"))
	})
	mux.HandleFunc("/oauth2/user_info", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		f.mu.Lock()
		defer f.mu.Unlock()
		id, ok := f.tokens[r.PostFormValue("access_token")]
		if !ok || r.PostFormValue("client_id") != testVKClientID {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":"invalid_token"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"user":{"user_id":%s,"first_name":"Test","last_name":"User","email":"%s"}}`, id.sub, id.email)
	})
	f.srv = httptest.NewServer(mux)
	return f
}

func (f *fakeVK) Close() { f.srv.Close() }

// addCode registers a one-time auth code (numeric sub!) and returns it with
// the device_id the client must echo to the exchange.
func (f *fakeVK) addCode(sub, email string) (code, deviceID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	code = fmt.Sprintf("vk-code-%d", f.n)
	f.codes[code] = vkCode{id: providerIdentity{sub: sub, email: email}, deviceID: testVKDeviceID}
	return code, testVKDeviceID
}
