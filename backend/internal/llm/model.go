package llm

// DefaultModel is the Claude model the RU backend requests by default. The
// worker passes it straight to anthropic-sdk-go (which has the matching
// anthropic.ModelClaudeOpus4_7 constant). Kept on the RU side so the model
// choice lives in one place and travels in the request payload (ARCH §11.3).
//
// Verify the current Opus id via ctx7 before bumping (CLAUDE.md "что обязательно
// сверять").
const DefaultModel = "claude-opus-4-7"
