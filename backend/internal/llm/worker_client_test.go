package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sseHandler(events []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flusher", http.StatusInternalServerError)
			return
		}
		for _, chunk := range events {
			if _, err := fmt.Fprint(w, chunk); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func TestWorkerClient_Send_ParsesEventsInOrder(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(sseHandler([]string{
		"event: message_started\ndata: {\"message_id\":\"m1\"}\n\n",
		"event: delta\ndata: {\"text\":\"hello\"}\n\n",
		"event: delta\ndata: {\"text\":\"world\"}\n\n",
		"event: usage\ndata: {\"tokens_in\":1,\"tokens_out\":2}\n\n",
		"event: done\ndata: {}\n\n",
	}))
	defer srv.Close()

	c, err := NewWorkerClient(&Config{Kind: "worker", BaseURL: srv.URL})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := c.Send(ctx, SendRequest{Model: "echo"})
	require.NoError(t, err)

	gotTypes := make([]EventType, 0, 5)
	for ev := range ch {
		gotTypes = append(gotTypes, ev.Type)
	}
	assert.Equal(t, []EventType{
		EventMessageStarted, EventDelta, EventDelta, EventUsage, EventDone,
	}, gotTypes)
}

func TestWorkerClient_Send_NonOKReturnsError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad input"))
	}))
	defer srv.Close()

	c, err := NewWorkerClient(&Config{BaseURL: srv.URL})
	require.NoError(t, err)

	_, err = c.Send(context.Background(), SendRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
}

func TestWorkerClient_Send_CancelClosesChannel(t *testing.T) {
	t.Parallel()
	// Хэндлер шлёт первое событие, ждёт долго перед вторым.
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		_, _ = fmt.Fprint(w, "event: delta\ndata: {\"text\":\"first\"}\n\n")
		flusher.Flush()
		<-block
	}))
	defer srv.Close()
	defer close(block)

	c, err := NewWorkerClient(&Config{BaseURL: srv.URL})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := c.Send(ctx, SendRequest{})
	require.NoError(t, err)

	first, ok := <-ch
	require.True(t, ok)
	assert.Equal(t, EventDelta, first.Type)

	cancel()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("channel did not close after cancel")
		}
	}
}

func TestNewWorkerClient_RequiresBaseURL(t *testing.T) {
	t.Parallel()
	_, err := NewWorkerClient(&Config{Kind: "worker"})
	require.Error(t, err)
}

func TestNewWorkerClient_MTLSRequiresPaths(t *testing.T) {
	t.Parallel()
	_, err := NewWorkerClient(&Config{
		Kind:        "worker",
		BaseURL:     "https://worker.example",
		MTLSEnabled: true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cert/key/CA path is empty")
}
