package llm

// DefaultModel is the Claude model the RU backend requests by default. The
// worker passes it straight to anthropic-sdk-go (which has the matching
// anthropic.ModelClaudeOpus4_7 constant). Kept on the RU side so the model
// choice lives in one place and travels in the request payload (ARCH §11.3).
//
// Verify the current Opus id via ctx7 before bumping (CLAUDE.md "что обязательно
// сверять").
const DefaultModel = "claude-opus-4-7"

// modelPricesPerMTok holds approximate Anthropic list prices in USD per million
// tokens (input, output). Used only to derive the claude_cost_usd metric — not
// billing. Update alongside the model id when pricing or DefaultModel changes.
var modelPricesPerMTok = map[string]struct{ in, out float64 }{
	"claude-opus-4-7": {in: 15, out: 75},
}

// EstimateCostUSD approximates the USD cost of a turn from its token counts.
// Unknown models fall back to DefaultModel's price so the metric never silently
// reads zero after a model bump.
func EstimateCostUSD(model string, tokensIn, tokensOut int64) float64 {
	p, ok := modelPricesPerMTok[model]
	if !ok {
		p = modelPricesPerMTok[DefaultModel]
	}
	return (float64(tokensIn)*p.in + float64(tokensOut)*p.out) / 1_000_000
}
