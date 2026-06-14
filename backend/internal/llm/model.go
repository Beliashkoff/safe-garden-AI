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
// tokens — the base (uncached) input and output rates. Cache reads and writes
// are derived from the input rate via the multipliers below. Used only to derive
// the claude_cost_usd metric — not billing. Update alongside DefaultModel when
// pricing changes.
var modelPricesPerMTok = map[string]struct{ in, out float64 }{
	"claude-opus-4-7": {in: 5, out: 25},
}

// Prompt-caching price multipliers relative to the base input rate (Anthropic
// pricing): a cache read costs 0.1x input, and writing the 5-minute ephemeral
// cache costs 1.25x input. The worker caches with the default 5-minute TTL
// (anthropic.NewCacheControlEphemeralParam), so cache creation is priced at
// 1.25x; switching any breakpoint to a 1-hour TTL would be 2x and require
// splitting cache-creation tokens by TTL.
const (
	cacheReadMultiplier  = 0.10
	cacheWriteMultiplier = 1.25
)

// TokenUsage is a turn's token counts split by how each input token is priced:
// uncached input, prompt-cache writes, and prompt-cache reads (ARCH §7.3). The
// model reports these separately; collapsing them would overcharge cache reads
// 10x and undercharge cache writes.
type TokenUsage struct {
	InputTokens      int64 // uncached input — full input rate
	CacheWriteTokens int64 // tokens written to the prompt cache — 1.25x input
	CacheReadTokens  int64 // tokens served from the prompt cache — 0.1x input
	OutputTokens     int64
}

// TotalInputTokens is every input token regardless of cache tier — the figure
// shown as the message's input token count (usage_log.tokens_in).
func (u TokenUsage) TotalInputTokens() int64 {
	return u.InputTokens + u.CacheWriteTokens + u.CacheReadTokens
}

// EstimateCostUSD approximates the USD cost of a turn, pricing cached input
// tokens at their reduced rates. Unknown models fall back to DefaultModel's
// price so the metric never silently reads zero after a model bump.
func EstimateCostUSD(model string, u TokenUsage) float64 {
	p, ok := modelPricesPerMTok[model]
	if !ok {
		p = modelPricesPerMTok[DefaultModel]
	}
	cost := float64(u.InputTokens)*p.in +
		float64(u.CacheWriteTokens)*p.in*cacheWriteMultiplier +
		float64(u.CacheReadTokens)*p.in*cacheReadMultiplier +
		float64(u.OutputTokens)*p.out
	return cost / 1_000_000
}
