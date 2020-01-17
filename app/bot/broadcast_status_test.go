package bot

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestBroadcastStatusBlinks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := NewBroadcastStatus(ctx, "", time.Millisecond, 5*time.Millisecond)
	b.ping = func(ctx context.Context, _ string) bool {
		return true
	}
	time.Sleep(time.Millisecond * 3)
	require.True(t, b.status) // switched to true
	b.ping = func(ctx context.Context, _ string) bool {
		return false
	}
	time.Sleep(time.Millisecond * 3)
	require.True(t, b.status) // still true, no deadline
	b.ping = func(ctx context.Context, _ string) bool {
		return true
	}
	time.Sleep(time.Millisecond * 3)
	require.True(t, b.status) // still true, no deadline

	time.Sleep(time.Millisecond * 5)
	require.True(t, b.status) // false, deadline deached
}

func TestBroadcastTransitions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b := NewBroadcastStatus(ctx, "", time.Millisecond, 5*time.Millisecond)
	b.ping = func(ctx context.Context, _ string) bool {
		return true
	}
	require.False(t, b.status) // at start status is off
	time.Sleep(time.Millisecond * 3)
	require.True(t, b.status) // switched to true
	b.ping = func(ctx context.Context, _ string) bool {
		return false
	}
	time.Sleep(time.Millisecond * 3)
	require.True(t, b.status) // still true, no deadline

	time.Sleep(time.Millisecond * 5)
	require.False(t, b.status) // false, deadline deached
	time.Sleep(time.Millisecond * 5)
	require.False(t, b.status) // still false
}

func TestBroadcastStatusBotOnMessage(t *testing.T) {
	b := &BroadcastStatus{}
	b.status = false
	resp, _ := b.OnMessage(Message{})
	require.Equal(t, "", resp)

	b.status = true
	resp, _ = b.OnMessage(Message{})
	require.Equal(t, MsgBroadcastStarted, resp)
	resp, _ = b.OnMessage(Message{})
	require.Equal(t, "", resp)

	b.status = false
	resp, _ = b.OnMessage(Message{})
	require.Equal(t, MsgBroadcastFinished, resp)
}

func TestPing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.True(t, ping(ctx, "http://ya.ru"))
	require.False(t, ping(ctx, "http://not-existing-url"))
	require.False(t, ping(ctx, "bad-url"))
	require.False(t, ping(nil, "http://ya.ru"))
}
