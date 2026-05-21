package llm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockClient_DefaultFixture(t *testing.T) {
	t.Parallel()
	c := NewMockClient()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := c.Send(ctx, SendRequest{})
	require.NoError(t, err)

	got := make([]EventType, 0, 6)
	for ev := range ch {
		got = append(got, ev.Type)
	}

	assert.Equal(t, []EventType{
		EventMessageStarted, EventDelta, EventDelta, EventDelta, EventUsage, EventDone,
	}, got)
}

func TestMockClient_CancelStopsStream(t *testing.T) {
	t.Parallel()
	c := NewMockClient()
	c.DelayPerEvent = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := c.Send(ctx, SendRequest{})
	require.NoError(t, err)

	// Получаем одно событие и отменяем.
	first, ok := <-ch
	require.True(t, ok)
	assert.Equal(t, EventMessageStarted, first.Type)
	cancel()

	// Канал должен быстро закрыться без новых событий.
	deadline := time.After(500 * time.Millisecond)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return // success
			}
		case <-deadline:
			t.Fatal("channel did not close after cancel")
		}
	}
}

func TestMockClient_CustomFixtureByModel(t *testing.T) {
	t.Parallel()
	c := &MockClient{
		Fixtures: map[string][]StreamEvent{
			"custom":  {{Type: EventDone}},
			"default": {{Type: EventError}},
		},
	}

	ctx := context.Background()
	ch, err := c.Send(ctx, SendRequest{Model: "custom"})
	require.NoError(t, err)

	got := make([]EventType, 0, 1)
	for ev := range ch {
		got = append(got, ev.Type)
	}
	assert.Equal(t, []EventType{EventDone}, got)
}

func TestMockClient_UnknownModelFallsBackToDefault(t *testing.T) {
	t.Parallel()
	c := &MockClient{
		Fixtures: map[string][]StreamEvent{
			"default": {{Type: EventDone}},
		},
	}
	ctx := context.Background()
	ch, err := c.Send(ctx, SendRequest{Model: "missing"})
	require.NoError(t, err)

	ev := <-ch
	assert.Equal(t, EventDone, ev.Type)
}
