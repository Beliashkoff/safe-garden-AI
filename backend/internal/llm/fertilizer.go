package llm

// FertilizerProduct is one catalog item returned by the recommend_fertilizer
// tool callback and streamed to the client in the fertilizer_card SSE event
// (ARCH §6.4, §11). Every field is non-PII catalog data, safe to cross the
// worker↔backend boundary and to persist in message_blocks.metadata.
type FertilizerProduct struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	ShortDesc   string `json:"short_desc"`
	ImageURL    string `json:"image_url"`
	DeeplinkURL string `json:"deeplink_url"`
}

// FertilizerToolArgs are the recommend_fertilizer tool inputs Claude produces
// (ARCH §7.2). Only Problem is required; the rest narrow the match.
type FertilizerToolArgs struct {
	Problem  string `json:"problem"`
	Plant    string `json:"plant,omitempty"`
	Severity string `json:"severity,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

// FertilizerToolRequest / FertilizerToolResponse are the wire contract for the
// worker→backend callback POST /internal/v1/tools/fertilizer (ARCH §11).
type FertilizerToolRequest struct {
	Args FertilizerToolArgs `json:"args"`
}

// FertilizerToolResponse carries 0–3 products. Empty Products means "no suitable
// fertilizer" — the worker then skips the fertilizer_card event and Claude
// answers with text only (SPEC Q2, empty-catalog behaviour).
type FertilizerToolResponse struct {
	Products []FertilizerProduct `json:"products"`
}

// FertilizerCardEvent is the payload of the fertilizer_card SSE event and of the
// fertilizer_card message block metadata (ARCH §4.3, §6.1).
type FertilizerCardEvent struct {
	Products []FertilizerProduct `json:"products"`
}
