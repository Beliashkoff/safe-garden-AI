package internalapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
)

type fakeRecommender struct {
	products []llm.FertilizerProduct
	err      error
	gotArgs  llm.FertilizerToolArgs
}

func (f *fakeRecommender) Recommend(_ context.Context, args llm.FertilizerToolArgs) ([]llm.FertilizerProduct, error) {
	f.gotArgs = args
	return f.products, f.err
}

func newTestServer(rec recommender) *httptest.Server {
	h := New(rec, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return httptest.NewServer(h.Routes())
}

func post(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url+"/internal/v1/tools/fertilizer", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	return resp
}

func TestPostFertilizer_ReturnsProducts(t *testing.T) {
	rec := &fakeRecommender{products: []llm.FertilizerProduct{
		{ID: "1", Slug: "k-boost", Name: "Калий-Буст", ShortDesc: "desc"},
	}}
	srv := newTestServer(rec)
	defer srv.Close()

	resp := post(t, srv.URL, `{"args":{"problem":"potassium_deficiency","plant":"tomato"}}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var out llm.FertilizerToolResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Products) != 1 || out.Products[0].Slug != "k-boost" {
		t.Fatalf("unexpected products: %+v", out.Products)
	}
	if rec.gotArgs.Problem != "potassium_deficiency" || rec.gotArgs.Plant != "tomato" {
		t.Fatalf("recommender got wrong args: %+v", rec.gotArgs)
	}
}

func TestPostFertilizer_EmptyCatalogReturnsEmptyArray(t *testing.T) {
	srv := newTestServer(&fakeRecommender{products: nil})
	defer srv.Close()

	resp := post(t, srv.URL, `{"args":{"problem":"wilting"}}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	// Must serialize as [] not null so the client always sees a products array.
	if !bytes.Contains(body, []byte(`"products":[]`)) {
		t.Fatalf("expected empty products array, got: %s", body)
	}
}

func TestPostFertilizer_MissingProblemIs400(t *testing.T) {
	srv := newTestServer(&fakeRecommender{})
	defer srv.Close()

	resp := post(t, srv.URL, `{"args":{"plant":"tomato"}}`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestPostFertilizer_InvalidJSONIs400(t *testing.T) {
	srv := newTestServer(&fakeRecommender{})
	defer srv.Close()

	resp := post(t, srv.URL, `not json`)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}
