package llm

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// WorkerClient — реализация Client поверх HTTP+SSE к llm-worker'у (ARCH §11.3).
// В prod использует mTLS: TLS-termination делает Caddy перед worker'ом, а
// клиент подсовывает свой сертификат и проверяет серверный по CA.
type WorkerClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewWorkerClient собирает клиента согласно конфигу. Если MTLSEnabled —
// читает cert/key/CA с диска и кладёт в http.Transport.TLSClientConfig.
// Если выключен — обычный http.Client (dev/локалка).
func NewWorkerClient(cfg *Config) (*WorkerClient, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("llm.worker: LLM_WORKER_BASE_URL is empty")
	}

	transport := &http.Transport{
		ResponseHeaderTimeout: 0, // стрим без таймаута, отменяем через ctx
	}

	if cfg.MTLSEnabled {
		tlsCfg, err := buildMTLSConfig(cfg)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = tlsCfg
	}

	return &WorkerClient{
		httpClient: &http.Client{Transport: transport},
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
	}, nil
}

func buildMTLSConfig(cfg *Config) (*tls.Config, error) {
	if cfg.MTLSCertPath == "" || cfg.MTLSKeyPath == "" || cfg.MTLSCAPath == "" {
		return nil, fmt.Errorf("llm.worker: mTLS enabled but cert/key/CA path is empty")
	}

	cert, err := tls.LoadX509KeyPair(cfg.MTLSCertPath, cfg.MTLSKeyPath)
	if err != nil {
		return nil, fmt.Errorf("llm.worker: load client cert: %w", err)
	}

	caPEM, err := os.ReadFile(cfg.MTLSCAPath)
	if err != nil {
		return nil, fmt.Errorf("llm.worker: read CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("llm.worker: CA file %s contains no certificates", cfg.MTLSCAPath)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// Send открывает SSE-стрим к worker'у и пушит события в канал. Канал
// закрывается при поступлении EventDone/EventError, при ошибке транспорта
// или при отмене контекста.
func (c *WorkerClient) Send(ctx context.Context, req SendRequest) (<-chan StreamEvent, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("llm.worker: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/llm/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm.worker: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm.worker: do request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("llm.worker: status %d: %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamEvent)
	go c.parseStream(ctx, resp.Body, ch)
	return ch, nil
}

func (c *WorkerClient) parseStream(ctx context.Context, body io.ReadCloser, ch chan<- StreamEvent) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	// SSE-сообщения могут быть большими (картинки base64 → дельты),
	// дефолтные 64 КБ маловаты.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventName string
	var dataLines []string

	flush := func() {
		if eventName == "" && len(dataLines) == 0 {
			return
		}
		ev := StreamEvent{Type: EventType(eventName), Data: json.RawMessage(strings.Join(dataLines, "\n"))}
		select {
		case ch <- ev:
		case <-ctx.Done():
		}
		eventName = ""
		dataLines = nil
	}

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
		// Прочие строки (включая SSE-комментарии вида ": heartbeat") игнорируем.
	}
	// Финальный flush на случай, если стрим оборвался без пустой строки.
	flush()
}
