package llmworker

import (
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testProvider(maxTokens int, override string) *anthropicProvider {
	return &anthropicProvider{
		maxTokens:     int64(maxTokens),
		modelOverride: override,
		logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// marshalParams renders the SDK params to JSON so we can assert the wire shape
// without introspecting opaque param types.
func marshalParams(t *testing.T, p *anthropicProvider, req messageRequest) string {
	t.Helper()
	b, err := json.Marshal(p.buildParams(req))
	require.NoError(t, err)
	return string(b)
}

func TestBuildParams_ModelSelection(t *testing.T) {
	req := messageRequest{
		Messages: []messageItem{{Role: "user", Content: []contentBlock{{Type: "text", Text: "hi"}}}},
	}

	// Empty req.Model + no override → DefaultModel.
	assert.Contains(t, marshalParams(t, testProvider(2048, ""), req), `"model":"claude-opus-4-7"`)

	// req.Model honoured when set.
	req.Model = "claude-sonnet-4-6"
	assert.Contains(t, marshalParams(t, testProvider(2048, ""), req), `"model":"claude-sonnet-4-6"`)

	// Override beats payload.
	assert.Contains(t, marshalParams(t, testProvider(2048, "claude-haiku-4-5"), req), `"model":"claude-haiku-4-5"`)
}

func TestBuildParams_SystemAndMetadataAndMaxTokens(t *testing.T) {
	req := messageRequest{
		System:   "Ты агроном.",
		Messages: []messageItem{{Role: "user", Content: []contentBlock{{Type: "text", Text: "hi"}}}},
		Metadata: requestMeta{UIDHash: "abc123", RequestID: "req_1"},
	}
	out := marshalParams(t, testProvider(1234, ""), req)

	assert.Contains(t, out, `"max_tokens":1234`)
	assert.Contains(t, out, "агроном")
	assert.Contains(t, out, `"cache_control"`, "system block must be cache-controlled (ephemeral)")
	assert.Contains(t, out, `"user_id":"abc123"`, "only the uid_hash crosses to Anthropic")
}

func TestBuildParams_SkipsEmptyContentMessages(t *testing.T) {
	req := messageRequest{
		Messages: []messageItem{
			{Role: "user", Content: []contentBlock{{Type: "text", Text: "hi"}}},
			{Role: "assistant", Content: []contentBlock{{Type: "text", Text: "ok"}}},
			{Role: "user", Content: []contentBlock{{Type: "image", MediaB64: "..."}}}, // no text → skipped in 2.2
		},
	}
	out := marshalParams(t, testProvider(2048, ""), req)
	assert.Equal(t, 2, strings.Count(out, `"role":`), "only the two text messages are sent")
}

func TestBuildParams_ToolsConverted(t *testing.T) {
	req := messageRequest{
		Messages: []messageItem{{Role: "user", Content: []contentBlock{{Type: "text", Text: "hi"}}}},
		Tools: []toolDef{{
			Name:        "recommend_fertilizer",
			Description: "Подбирает удобрение",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"problem":{"type":"string"}},"required":["problem"]}`),
		}},
	}
	out := marshalParams(t, testProvider(2048, ""), req)
	assert.Contains(t, out, "recommend_fertilizer")
	assert.Contains(t, out, `"problem"`)
	assert.Contains(t, out, `"cache_control"`, "last tool is cache-controlled")
}
