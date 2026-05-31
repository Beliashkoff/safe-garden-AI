package audio

import "context"

// MockTranscriber is a network-free transcriber for dev and unit tests. It
// returns a fixed transcript, mirroring llm.MockClient.
type MockTranscriber struct {
	Text       string
	DurationMs int64
}

func NewMockTranscriber() *MockTranscriber {
	return &MockTranscriber{Text: "мок-транскрипция голосового сообщения", DurationMs: 1000}
}

var _ Transcriber = (*MockTranscriber)(nil)

func (m *MockTranscriber) Transcribe(ctx context.Context, _ []byte, _ string) (Result, error) {
	select {
	case <-ctx.Done():
		return Result{}, ctx.Err()
	default:
		return Result{Text: m.Text, DurationMs: m.DurationMs}, nil
	}
}
