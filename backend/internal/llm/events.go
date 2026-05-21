package llm

// EventType — тип события в SSE-стриме между worker'ом и бэкендом
// (контракт ARCH §11.3). Используется и сервером (worker), и клиентом —
// одна точка истины, чтобы стороны не разъезжались по строковым литералам.
type EventType string

const (
	EventMessageStarted EventType = "message_started"
	EventDelta          EventType = "delta"
	EventToolUse        EventType = "tool_use"
	EventFertilizerCard EventType = "fertilizer_card"
	EventUsage          EventType = "usage"
	EventError          EventType = "error"
	EventDone           EventType = "done"
)
