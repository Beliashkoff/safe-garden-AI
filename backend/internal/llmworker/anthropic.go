package llmworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
)

// maxToolTurns bounds the tool-use loop so a misbehaving model (or a tool that
// keeps prompting another call) can never stream forever.
const maxToolTurns = 4

// fertilizerRecommender resolves a recommend_fertilizer tool call into catalog
// products via the RU backend (ARCH §11). nil when the callback is unconfigured
// (dev) — the tool then reports "catalog unavailable" and Claude answers text.
type fertilizerRecommender interface {
	Recommend(ctx context.Context, args json.RawMessage) ([]llm.FertilizerProduct, error)
}

// anthropicProvider streams completions from Claude via anthropic-sdk-go. This
// is the ONLY place in the repo that imports the SDK (CLAUDE.md invariant #5).
// It runs only on the Frankfurt worker; the RU backend never reaches Anthropic.
type anthropicProvider struct {
	client        anthropic.Client
	maxTokens     int64
	modelOverride string
	fertilizer    fertilizerRecommender
	logger        *slog.Logger
}

func newAnthropicProvider(apiKey string, maxTokens int, modelOverride string, fertilizer fertilizerRecommender, logger *slog.Logger) *anthropicProvider {
	return &anthropicProvider{
		client:        anthropic.NewClient(option.WithAPIKey(apiKey)),
		maxTokens:     int64(maxTokens),
		modelOverride: modelOverride,
		fertilizer:    fertilizer,
		logger:        logger,
	}
}

// stream runs the Claude completion as a multi-turn tool-use loop (ARCH §11):
// it streams text deltas live, and whenever Claude stops with a tool_use it
// resolves the tool via the backend callback, emits the fertilizer_card event,
// feeds the tool_result back, and continues — until Claude produces a final
// answer or the turn budget is exhausted.
func (p *anthropicProvider) stream(ctx context.Context, req messageRequest, sink eventSink) error {
	params := p.buildParams(req)
	var totalIn, totalOut int64
	startedSent := false

	for turn := 0; turn < maxToolTurns; turn++ {
		stream := p.client.Messages.NewStreaming(ctx, params)

		acc := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			if err := acc.Accumulate(event); err != nil {
				p.logger.ErrorContext(ctx, "accumulate stream event failed", "err", err.Error())
			}
			switch e := event.AsAny().(type) {
			case anthropic.MessageStartEvent:
				// Emit message_started only once across the whole loop.
				if !startedSent {
					startedSent = true
					if err := sink.started(e.Message.ID); err != nil {
						return err
					}
				}
			case anthropic.ContentBlockDeltaEvent:
				if e.Delta.Text != "" {
					if err := sink.delta(e.Delta.Text); err != nil {
						return err
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			// Log the real error server-side; the client gets a generic code with
			// no upstream/PII detail.
			p.logger.ErrorContext(ctx, "claude stream error", "err", err.Error())
			sink.failed("upstream_error", "the model service is temporarily unavailable")
			return err
		}

		totalIn += acc.Usage.InputTokens + acc.Usage.CacheCreationInputTokens + acc.Usage.CacheReadInputTokens
		totalOut += acc.Usage.OutputTokens

		if acc.StopReason != anthropic.StopReasonToolUse {
			// Final answer — no (more) tool calls.
			if err := sink.usage(totalIn, totalOut); err != nil {
				return err
			}
			return sink.done()
		}

		// Tool turn: carry the assistant's tool_use message, then answer each
		// tool_use with a tool_result and loop.
		params.Messages = append(params.Messages, acc.ToParam())
		toolResults := make([]anthropic.ContentBlockParamUnion, 0, len(acc.Content))
		for _, block := range acc.Content {
			tu := block.AsToolUse()
			if tu.Name == "" {
				continue
			}
			resultText, isErr := p.handleTool(ctx, tu, sink)
			toolResults = append(toolResults, anthropic.NewToolResultBlock(tu.ID, resultText, isErr))
		}
		params.Messages = append(params.Messages, anthropic.NewUserMessage(toolResults...))
	}

	// Turn budget exhausted (model kept asking for tools). Report usage so far
	// and surface a non-fatal error rather than hanging.
	_ = sink.usage(totalIn, totalOut)
	sink.failed("tool_loop_exhausted", "the assistant could not complete the request")
	return nil
}

// handleTool dispatches a single tool_use, returning the tool_result text to
// feed back to Claude and whether it is an error result.
func (p *anthropicProvider) handleTool(ctx context.Context, tu anthropic.ToolUseBlock, sink eventSink) (string, bool) {
	// Surface the call to the client (telemetry/contract, ARCH §4.3). Args are
	// the user's own diagnosis, not PII.
	_ = sink.toolUse(tu.Name, tu.Input)

	switch tu.Name {
	case llm.RecommendFertilizerName:
		return p.handleRecommendFertilizer(ctx, tu.Input, sink)
	default:
		p.logger.WarnContext(ctx, "unknown tool requested", "tool", tu.Name)
		return "Unknown tool.", true
	}
}

// handleRecommendFertilizer calls the backend catalog, emits the fertilizer_card
// event when products are found, and returns a compact text result so Claude can
// reference the products by name. Empty catalog → no card, text-only answer.
func (p *anthropicProvider) handleRecommendFertilizer(ctx context.Context, input json.RawMessage, sink eventSink) (string, bool) {
	if p.fertilizer == nil {
		p.logger.WarnContext(ctx, "recommend_fertilizer called but callback is unconfigured")
		return "Каталог удобрений недоступен.", false
	}

	products, err := p.fertilizer.Recommend(ctx, input)
	if err != nil {
		p.logger.ErrorContext(ctx, "fertilizer callback failed", "err", err.Error())
		return "Не удалось получить рекомендации из каталога.", false
	}
	if len(products) == 0 {
		return "Подходящего удобрения в каталоге нет.", false
	}

	if data, err := json.Marshal(llm.FertilizerCardEvent{Products: products}); err == nil {
		_ = sink.fertilizerCard(data)
	} else {
		p.logger.ErrorContext(ctx, "marshal fertilizer card failed", "err", err.Error())
	}

	return formatProductsForClaude(products), false
}

// formatProductsForClaude renders the products as a short text block for the
// tool_result so Claude can mention them in its reply (the structured card is
// delivered separately via the fertilizer_card event).
func formatProductsForClaude(products []llm.FertilizerProduct) string {
	var b strings.Builder
	b.WriteString("Найдены подходящие удобрения из каталога:\n")
	for i, pr := range products {
		fmt.Fprintf(&b, "%d. %s — %s", i+1, pr.Name, pr.ShortDesc)
		// Price travels in the tool_result so Claude can answer a direct price
		// question (system prompt: state it only if the user asks). It is also
		// shown in the structured card; omitting it here would make the prompt's
		// price rule unsatisfiable.
		if pr.PriceRub != nil {
			fmt.Fprintf(&b, " (цена: %d ₽)", *pr.PriceRub)
		}
		b.WriteByte('\n')
	}
	b.WriteString("Карточки уже показаны пользователю. Кратко порекомендуй их в ответе. " +
		"Цену называй только если пользователь спросил.")
	return b.String()
}

// buildParams converts the neutral wire request into anthropic params. Pure (no
// I/O) so it is unit-tested directly.
func (p *anthropicProvider) buildParams(req messageRequest) anthropic.MessageNewParams {
	model := req.Model
	if p.modelOverride != "" {
		model = p.modelOverride
	}
	if model == "" {
		model = llm.DefaultModel
	}

	params := anthropic.MessageNewParams{
		Model:     model, // anthropic.Model is a string alias
		MaxTokens: p.maxTokens,
		Messages:  toAnthropicMessages(req.Messages),
	}

	// System prompt with ephemeral cache_control (ARCH §7.3 prompt caching).
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{
			Text:         req.System,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}}
	}

	// Only the anonymized uid_hash crosses to Anthropic (ARCH §11.4).
	if req.Metadata.UIDHash != "" {
		params.Metadata = anthropic.MetadataParam{UserID: anthropic.String(req.Metadata.UIDHash)}
	}

	if tools := toAnthropicTools(req.Tools); len(tools) > 0 {
		params.Tools = tools
	}
	return params
}

func toAnthropicMessages(items []messageItem) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(items))
	for _, m := range items {
		blocks := make([]anthropic.ContentBlockParamUnion, 0, len(m.Content))
		for _, b := range m.Content {
			// Stage 3.1 adds base64 image blocks. The RU backend has already
			// converted HEIC→JPEG and validated the media type; the worker only
			// forwards. audio/tool_result blocks land in Stage 4+.
			switch {
			case b.Type == "text" && b.Text != "":
				blocks = append(blocks, anthropic.NewTextBlock(b.Text))
			case b.Type == "image" && b.MediaB64 != "" && b.MediaType != "":
				blocks = append(blocks, anthropic.NewImageBlockBase64(b.MediaType, b.MediaB64))
			}
		}
		if len(blocks) == 0 {
			continue
		}
		if m.Role == "assistant" {
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		} else {
			out = append(out, anthropic.NewUserMessage(blocks...))
		}
	}
	return out
}

func toAnthropicTools(tools []toolDef) []anthropic.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for i, td := range tools {
		var schema struct {
			Properties any      `json:"properties"`
			Required   []string `json:"required"`
		}
		_ = json.Unmarshal(td.InputSchema, &schema)

		tp := anthropic.ToolParam{
			Name:        td.Name,
			Description: anthropic.String(td.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: schema.Properties,
				Required:   schema.Required,
			},
		}
		// Cache the tool definitions (they are stable across turns) by marking
		// the last tool block ephemeral (ARCH §7.3).
		if i == len(tools)-1 {
			tp.CacheControl = anthropic.NewCacheControlEphemeralParam()
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: &tp})
	}
	return out
}
