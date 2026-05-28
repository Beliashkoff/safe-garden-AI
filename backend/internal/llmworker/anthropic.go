package llmworker

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/Beliashkoff/safe-garden-AI/backend/internal/llm"
)

// anthropicProvider streams completions from Claude via anthropic-sdk-go. This
// is the ONLY place in the repo that imports the SDK (CLAUDE.md invariant #5).
// It runs only on the Frankfurt worker; the RU backend never reaches Anthropic.
type anthropicProvider struct {
	client        anthropic.Client
	maxTokens     int64
	modelOverride string
	logger        *slog.Logger
}

func newAnthropicProvider(apiKey string, maxTokens int, modelOverride string, logger *slog.Logger) *anthropicProvider {
	return &anthropicProvider{
		client:        anthropic.NewClient(option.WithAPIKey(apiKey)),
		maxTokens:     int64(maxTokens),
		modelOverride: modelOverride,
		logger:        logger,
	}
}

func (p *anthropicProvider) stream(ctx context.Context, req messageRequest, sink eventSink) error {
	stream := p.client.Messages.NewStreaming(ctx, p.buildParams(req))

	acc := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := acc.Accumulate(event); err != nil {
			p.logger.ErrorContext(ctx, "accumulate stream event failed", "err", err.Error())
		}
		switch e := event.AsAny().(type) {
		case anthropic.MessageStartEvent:
			if err := sink.started(e.Message.ID); err != nil {
				return err
			}
		case anthropic.ContentBlockStartEvent:
			if tu := e.ContentBlock.AsToolUse(); tu.Name != "" {
				if err := sink.toolUse(tu.Name, tu.Input); err != nil {
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

	in := acc.Usage.InputTokens + acc.Usage.CacheCreationInputTokens + acc.Usage.CacheReadInputTokens
	if err := sink.usage(in, acc.Usage.OutputTokens); err != nil {
		return err
	}
	return sink.done()
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
			// Stage 2.2 is text-only; image/audio/tool_result blocks land in
			// Stage 3+ (the wire schema already carries them).
			if b.Type == "text" && b.Text != "" {
				blocks = append(blocks, anthropic.NewTextBlock(b.Text))
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
