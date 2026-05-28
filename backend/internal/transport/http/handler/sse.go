package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// writeSSE serializes data as JSON and writes one `event:/data:` SSE block,
// flushing immediately. Returns an error if the writer cannot flush or the
// client has gone away (so the caller stops streaming).
func writeSSE(w http.ResponseWriter, event string, data any) error {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("sse: response writer does not support flushing")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("sse: marshal %s: %w", event, err)
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return fmt.Errorf("sse: write: %w", err)
	}
	flusher.Flush()
	return nil
}

// setSSEHeaders sets the standard SSE headers before the first write.
// X-Accel-Buffering disables proxy buffering (Caddy/nginx) so deltas stream.
func setSSEHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
}
