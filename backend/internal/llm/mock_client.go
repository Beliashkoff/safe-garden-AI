package llm

import (
	"context"
	"encoding/json"
	"time"
)

// MockClient — фикстурный клиент для dev и unit-тестов. Не использует сеть.
// По умолчанию отдаёт фиксированную последовательность событий — этого
// достаточно для проверки usecase-слоя и handlers'ов в Этапах 1–2.
type MockClient struct {
	// Fixtures — словарь "ключ → события". Ключ выбирается по Model
	// в SendRequest; если ключ не найден — используется "default".
	Fixtures map[string][]StreamEvent

	// DelayPerEvent — задержка между событиями (для тестов отмены).
	DelayPerEvent time.Duration
}

// NewMockClient возвращает клиент с одной дефолтной фикстурой.
func NewMockClient() *MockClient {
	return &MockClient{
		Fixtures: map[string][]StreamEvent{
			"default": defaultMockFixture(),
		},
	}
}

func (m *MockClient) Send(ctx context.Context, req SendRequest) (<-chan StreamEvent, error) {
	events, ok := m.Fixtures[req.Model]
	if !ok {
		events = m.Fixtures["default"]
	}

	ch := make(chan StreamEvent)
	go func() {
		defer close(ch)
		for _, ev := range events {
			if m.DelayPerEvent > 0 {
				select {
				case <-time.After(m.DelayPerEvent):
				case <-ctx.Done():
					return
				}
			}
			select {
			case ch <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, nil
}

func defaultMockFixture() []StreamEvent {
	mustJSON := func(v any) json.RawMessage {
		b, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		return b
	}
	return []StreamEvent{
		{Type: EventMessageStarted, Data: mustJSON(map[string]string{"message_id": "msg_mock_1"})},
		{Type: EventDelta, Data: mustJSON(map[string]string{"text": "Mock "})},
		{Type: EventDelta, Data: mustJSON(map[string]string{"text": "response "})},
		{Type: EventDelta, Data: mustJSON(map[string]string{"text": "from llm.MockClient"})},
		{Type: EventUsage, Data: mustJSON(map[string]int{"tokens_in": 0, "tokens_out": 0})},
		{Type: EventDone, Data: mustJSON(struct{}{})},
	}
}
