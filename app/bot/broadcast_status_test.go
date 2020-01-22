package bot

import (
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Kind of integration test to check all workflow
func TestBroadcastStatusTransitions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	status := true // hold here ping status
	statusMx := sync.Mutex{}
	setStatus := func(s bool) {
		statusMx.Lock()
		defer statusMx.Unlock()
		status = s
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		statusMx.Lock()
		defer statusMx.Unlock()
		if status {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	b := NewBroadcastStatus(ctx, BroadcastParams{
		Url:          ts.URL,
		PingInterval: time.Millisecond,
		DelayToOff:   100 * time.Millisecond,
		Client:       http.Client{},
	})

	// Test reacts on first message
	resp, _ := b.OnMessage(Message{})
	require.Equal(t, MsgBroadcastFinished, resp)

	// Test do not react on second message because status not changed
	resp, _ = b.OnMessage(Message{})
	require.Equal(t, "", resp)

	// Wait for off->on
	time.Sleep(20 * time.Millisecond)
	resp, _ = b.OnMessage(Message{})
	require.Equal(t, MsgBroadcastStarted, resp)
	require.True(t, b.getStatus())

	// off
	setStatus(false)
	// Still on, no deadline reached
	time.Sleep(20 * time.Millisecond)
	resp, _ = b.OnMessage(Message{})
	require.Equal(t, "", resp)
	require.True(t, b.getStatus())

	// Deadline reached on->off
	time.Sleep(110 * time.Millisecond)
	resp, _ = b.OnMessage(Message{})
	require.Equal(t, MsgBroadcastFinished, resp)
	require.False(t, b.getStatus())
}

func TestBroadcastStatusOffToOn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	b := &BroadcastStatus{}
	b.check(ctx, time.Time{}, BroadcastParams{
		Url:    ts.URL,
		Client: http.Client{},
	})

	require.True(t, b.status)
}

func TestBroadcastStatusOffToOff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	b := &BroadcastStatus{}
	b.status = false
	b.check(ctx, time.Time{}, BroadcastParams{
		Url:    ts.URL,
		Client: http.Client{},
	})

	require.False(t, b.status)
}

func TestBroadcastStatusOnToOffNoDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	b := &BroadcastStatus{}
	b.status = true
	b.check(ctx, time.Now(), BroadcastParams{
		Url:        ts.URL,
		DelayToOff: time.Second,
		Client:     http.Client{},
	})

	require.True(t, b.status)
}

func TestBroadcastStatusOnToOffWithDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	b := &BroadcastStatus{}
	b.status = true
	b.check(ctx, time.Now().Add(-2*time.Second), BroadcastParams{
		Url:        ts.URL,
		DelayToOff: time.Second,
		Client:     http.Client{},
	})

	require.False(t, b.status)
}

func TestFirstOnMessageReturnsCurrentState(t *testing.T) {
	b := &BroadcastStatus{}
	response, answer := b.OnMessage(Message{})
	require.True(t, answer)
	require.Equal(t, MsgBroadcastFinished, response)
}

func TestOnMessageReturnsNothingIfStateNotChanged(t *testing.T) {
	b := &BroadcastStatus{fistMsgSent: true}
	_, answer := b.OnMessage(Message{})
	require.False(t, answer)

	b = &BroadcastStatus{fistMsgSent: true, status: true, lastSentStatus: true}
	_, answer = b.OnMessage(Message{})
	require.False(t, answer)
}

func TestOnMessageReturnsReplyOnChange(t *testing.T) {
	b := &BroadcastStatus{fistMsgSent: true, lastSentStatus: false, status: true} // OFF ->ON
	resp, answer := b.OnMessage(Message{})
	require.True(t, answer)
	require.Equal(t, MsgBroadcastStarted, resp)

	b = &BroadcastStatus{fistMsgSent: true, lastSentStatus: true, status: false} // ON -> OFF
	resp, answer = b.OnMessage(Message{})
	require.True(t, answer)
	require.Equal(t, MsgBroadcastFinished, resp)
}

func TestPingReturnsTrueOn200Status(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	require.True(t, ping(ctx, http.Client{}, ts.URL))
}

func TestPingReturnsFalseOnNot200Status(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	require.False(t, ping(ctx, http.Client{}, ts.URL))
}

func TestPingReturnsFalseOnUnableToDoRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	require.False(t, ping(ctx, http.Client{}, "http://localhost:9873"))
}

func TestPingReturnsFalseOnUnableToCreateReq(t *testing.T) {
	require.False(t, ping(nil, http.Client{}, "http://localhost:9873"))
}

func TestReactOnAnyMessage(t *testing.T) {
	require.Empty(t, (&BroadcastStatus{}).ReactOn())
}
