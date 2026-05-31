package llmworker

import (
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
	"time"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
)

const fertilizerCallbackTimeout = 10 * time.Second

// fertilizerClient calls the RU backend's internal recommend_fertilizer endpoint
// over mTLS (ARCH §11). It mirrors internal/llm/worker_client.go but in the
// reverse direction (worker is the client). Only anonymized tool args cross the
// wire — no PII (CLAUDE.md invariant #5).
type fertilizerClient struct {
	httpClient *http.Client
	url        string
}

func newFertilizerClient(cfg *Config) (*fertilizerClient, error) {
	if cfg.BackendCallbackURL == "" {
		return nil, fmt.Errorf("llmworker: BACKEND_CALLBACK_URL is empty")
	}

	transport := &http.Transport{}
	if cfg.BackendCallbackMTLSEnabled {
		tlsCfg, err := buildCallbackMTLS(cfg)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = tlsCfg
	}

	return &fertilizerClient{
		httpClient: &http.Client{Transport: transport, Timeout: fertilizerCallbackTimeout},
		url:        strings.TrimRight(cfg.BackendCallbackURL, "/") + "/internal/v1/tools/fertilizer",
	}, nil
}

func buildCallbackMTLS(cfg *Config) (*tls.Config, error) {
	if cfg.BackendCallbackClientCertPath == "" || cfg.BackendCallbackClientKeyPath == "" || cfg.BackendCallbackCAPath == "" {
		return nil, fmt.Errorf("llmworker: callback mTLS enabled but cert/key/CA path is empty")
	}

	cert, err := tls.LoadX509KeyPair(cfg.BackendCallbackClientCertPath, cfg.BackendCallbackClientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("llmworker: load callback client cert: %w", err)
	}

	caPEM, err := os.ReadFile(cfg.BackendCallbackCAPath)
	if err != nil {
		return nil, fmt.Errorf("llmworker: read callback CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("llmworker: callback CA %s contains no certificates", cfg.BackendCallbackCAPath)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// Recommend posts the raw Claude tool input ({problem, plant?, severity?,
// notes?}) to the backend and returns the catalog products (0–3).
func (c *fertilizerClient) Recommend(ctx context.Context, args json.RawMessage) ([]llm.FertilizerProduct, error) {
	var toolArgs llm.FertilizerToolArgs
	if err := json.Unmarshal(args, &toolArgs); err != nil {
		return nil, fmt.Errorf("llmworker: bad tool args: %w", err)
	}

	body, err := json.Marshal(llm.FertilizerToolRequest{Args: toolArgs})
	if err != nil {
		return nil, fmt.Errorf("llmworker: marshal callback: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llmworker: build callback: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llmworker: callback request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("llmworker: callback status %d: %s", resp.StatusCode, string(b))
	}

	var out llm.FertilizerToolResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("llmworker: decode callback: %w", err)
	}
	return out.Products, nil
}
