package llmworker

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// writeSSE сериализует data в JSON и пишет SSE-блок `event: ...\ndata: ...\n\n`.
// Возвращает ошибку, если ResponseWriter не поддерживает Flusher.
func writeSSE(w http.ResponseWriter, event string, data any) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support flushing")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal sse data: %w", err)
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return fmt.Errorf("write sse: %w", err)
	}
	flusher.Flush()
	return nil
}

// setSSEHeaders выставляет стандартные заголовки SSE до первой записи.
func setSSEHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
}
