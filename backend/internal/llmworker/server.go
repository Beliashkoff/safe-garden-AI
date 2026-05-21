package llmworker

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
	"github.com/Beliashkoff/safe-garden-AI/backend/internal/observability"
)

// Server — HTTP-обёртка вокруг echo-handler'а. В Этапе 2.2 сюда добавятся
// anthropic-клиент и tool-callback к РФ-бэкенду.
type Server struct {
	logger *slog.Logger
	cfg    *Config
}

func New(cfg *Config, logger *slog.Logger) *Server {
	return &Server{cfg: cfg, logger: logger}
}

// Routes возвращает chi-роутер с применёнными middleware. Выделено отдельно,
// чтобы тесты могли поднять server через httptest.NewServer(srv.Routes()).
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(observability.AccessLog(s.logger))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Post("/v1/llm/messages", s.handleMessages)
	return r
}

// handleMessages — echo-реализация (Этап 0.7). На вход — payload по контракту
// ARCH §11.3, на выход — SSE с зеркальной last-user-message текстовой дельтой.
// В 2.2 эту функцию заменит вызов anthropic-sdk-go.
func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB лимит на echo
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "validation_failed", "read body: "+err.Error())
		return
	}
	defer r.Body.Close()

	var req messageRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json: "+err.Error())
		return
	}
	if err := req.validate(bodyBytes); err != nil {
		s.writeError(w, http.StatusBadRequest, "validation_failed", err.Error())
		return
	}

	setSSEHeaders(w)
	w.WriteHeader(http.StatusOK)

	messageID := "msg_echo_" + randomID()

	if err := writeSSE(w, string(llm.EventMessageStarted), map[string]string{"message_id": messageID}); err != nil {
		s.logger.WarnContext(r.Context(), "sse write failed", "err", err)
		return
	}

	text := req.lastUserText()
	if text == "" {
		text = "(empty)"
	}
	// Стримим дельты «по словам» — имитация поведения Claude.
	for _, word := range splitWords(text) {
		select {
		case <-r.Context().Done():
			s.logger.InfoContext(r.Context(), "client cancelled before done")
			return
		default:
		}
		if err := writeSSE(w, string(llm.EventDelta), map[string]string{"text": word}); err != nil {
			s.logger.WarnContext(r.Context(), "sse write failed", "err", err)
			return
		}
	}

	if err := writeSSE(w, string(llm.EventUsage), map[string]int{"tokens_in": 0, "tokens_out": 0}); err != nil {
		s.logger.WarnContext(r.Context(), "sse write failed", "err", err)
		return
	}
	if err := writeSSE(w, string(llm.EventDone), struct{}{}); err != nil {
		s.logger.WarnContext(r.Context(), "sse write failed", "err", err)
	}
}

func (s *Server) writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": msg},
	})
}

func splitWords(s string) []string {
	// Сохраняем пробелы как часть «следующего» токена, чтобы клиент мог
	// просто склеить дельты без дополнительной логики.
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return nil
	}
	out := make([]string, 0, len(fields))
	for i, f := range fields {
		if i == 0 {
			out = append(out, f)
		} else {
			out = append(out, " "+f)
		}
	}
	return out
}

func randomID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}
