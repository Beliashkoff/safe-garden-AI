package llmworker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeRecommender is a test fertilizerRecommender.
type fakeRecommender struct {
	products []llm.FertilizerProduct
	err      error
	gotArgs  json.RawMessage
}

func (f *fakeRecommender) Recommend(_ context.Context, args json.RawMessage) ([]llm.FertilizerProduct, error) {
	f.gotArgs = args
	return f.products, f.err
}

// captureSink records the events a provider emits (in-memory eventSink).
type captureSink struct {
	cards [][]byte
	tools []string
}

func (s *captureSink) started(string) error { return nil }
func (s *captureSink) delta(string) error   { return nil }
func (s *captureSink) toolUse(name string, _ json.RawMessage) error {
	s.tools = append(s.tools, name)
	return nil
}
func (s *captureSink) fertilizerCard(data json.RawMessage) error {
	b := make([]byte, len(data))
	copy(b, data)
	s.cards = append(s.cards, b)
	return nil
}
func (s *captureSink) usage(int64, int64) error { return nil }
func (s *captureSink) done() error              { return nil }
func (s *captureSink) failed(string, string)    {}

func fertProvider(rec fertilizerRecommender) *anthropicProvider {
	return &anthropicProvider{fertilizer: rec, logger: discardLogger()}
}

func TestHandleRecommendFertilizer_EmitsCardAndResult(t *testing.T) {
	rec := &fakeRecommender{products: []llm.FertilizerProduct{
		{ID: "1", Slug: "k-boost", Name: "Калий-Буст", ShortDesc: "Калийная подкормка"},
	}}
	p := fertProvider(rec)
	sink := &captureSink{}

	args := json.RawMessage(`{"problem":"potassium_deficiency","plant":"tomato"}`)
	result, isErr := p.handleRecommendFertilizer(context.Background(), args, sink)

	if isErr {
		t.Fatalf("expected non-error result")
	}
	if len(sink.cards) != 1 {
		t.Fatalf("expected 1 fertilizer_card emitted, got %d", len(sink.cards))
	}
	var ev llm.FertilizerCardEvent
	if err := json.Unmarshal(sink.cards[0], &ev); err != nil {
		t.Fatalf("card payload not valid FertilizerCardEvent: %v", err)
	}
	if len(ev.Products) != 1 || ev.Products[0].Slug != "k-boost" {
		t.Fatalf("unexpected card products: %+v", ev.Products)
	}
	if result == "" {
		t.Fatalf("expected non-empty tool_result text for Claude")
	}
	if string(rec.gotArgs) != string(args) {
		t.Fatalf("recommender received wrong args: %s", rec.gotArgs)
	}
}

func TestHandleRecommendFertilizer_EmptyCatalogNoCard(t *testing.T) {
	p := fertProvider(&fakeRecommender{products: nil})
	sink := &captureSink{}

	result, isErr := p.handleRecommendFertilizer(context.Background(), json.RawMessage(`{"problem":"wilting"}`), sink)

	if isErr {
		t.Fatalf("empty catalog must not be an error result")
	}
	if len(sink.cards) != 0 {
		t.Fatalf("expected no fertilizer_card when catalog is empty, got %d", len(sink.cards))
	}
	if result == "" {
		t.Fatalf("expected a text result telling Claude there is no match")
	}
}

func TestHandleRecommendFertilizer_NilRecommender(t *testing.T) {
	p := fertProvider(nil) // callback unconfigured (dev)
	sink := &captureSink{}

	result, isErr := p.handleRecommendFertilizer(context.Background(), json.RawMessage(`{"problem":"wilting"}`), sink)

	if isErr {
		t.Fatalf("unconfigured callback should degrade gracefully, not as tool error")
	}
	if len(sink.cards) != 0 {
		t.Fatalf("expected no card when recommender is nil")
	}
	if result == "" {
		t.Fatalf("expected a text result")
	}
}

func TestHandleRecommendFertilizer_CallbackErrorIsGraceful(t *testing.T) {
	p := fertProvider(&fakeRecommender{err: errors.New("backend down")})
	sink := &captureSink{}

	result, isErr := p.handleRecommendFertilizer(context.Background(), json.RawMessage(`{"problem":"wilting"}`), sink)

	if isErr {
		t.Fatalf("callback failure should be reported to Claude as text, not crash the stream")
	}
	if len(sink.cards) != 0 {
		t.Fatalf("expected no card on callback error")
	}
	if result == "" {
		t.Fatalf("expected a text result on callback error")
	}
}

func TestHandleTool_UnknownToolIsError(t *testing.T) {
	p := fertProvider(&fakeRecommender{})
	sink := &captureSink{}

	tu := anthropic.ToolUseBlock{Name: "does_not_exist", Input: json.RawMessage(`{}`)}
	result, isErr := p.handleTool(context.Background(), tu, sink)

	if !isErr {
		t.Fatalf("unknown tool must return an error tool_result")
	}
	if result == "" {
		t.Fatalf("expected a result string for the unknown tool")
	}
}
