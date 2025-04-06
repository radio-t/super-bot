package bot

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	// MsgBroadcastStarted defines text to be sent by the bot when the broadcast started
	MsgBroadcastStarted = "Вещание началось. Приобщиться можно тут: https://stream.radio-t.com/"

	// MsgBroadcastFinished defines text to be sent by the bot when the broadcast finished
	MsgBroadcastFinished = "Вещание завершилось"
)

// BroadcastParams defines parameters for broadcast detection
type BroadcastParams struct {
	URL          string        // URL for "ping"
	PingInterval time.Duration // ping interval
	DelayToOff   time.Duration // state will be switched to off in no ok replies from URL in this interval
	Client       http.Client   // http client
}

// BroadcastStatus bot replies with current broadcast status
type BroadcastStatus struct {
	status         bool // current broadcast status
	lastSentStatus bool // last status sent with OnMessage
	statusMx       sync.Mutex
}

// NewBroadcastStatus starts status checking goroutine and returns bot instance
func NewBroadcastStatus(ctx context.Context, params BroadcastParams) *BroadcastStatus {
	log.Printf("[INFO] BroadcastStatus bot with %v", params.URL)
	b := &BroadcastStatus{}
	go b.checker(ctx, params)
	return b
}

// Help returns help message
func (b *BroadcastStatus) Help() string {
	return ""
}

// OnMessage returns current broadcast status if it was changed
func (b *BroadcastStatus) OnMessage(_ Message) (response Response) {
	b.statusMx.Lock()
	defer b.statusMx.Unlock()

	response.Pin = false
	if b.lastSentStatus != b.status {
		response.Send = true
		if b.status {
			response.Text = MsgBroadcastStarted
		} else {
			response.Text = MsgBroadcastFinished
			response.Unpin = true // unpin message "broadcast started" (sent by outside clients)
		}
		b.lastSentStatus = b.status
	}
	return
}

func (b *BroadcastStatus) checker(ctx context.Context, params BroadcastParams) {
	lastOn := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(params.PingInterval):
			lastOn = b.check(ctx, lastOn, params)
		}
	}
}

// check do ping to url and change current state
func (b *BroadcastStatus) check(ctx context.Context, lastOn time.Time, params BroadcastParams) time.Time {
	b.statusMx.Lock()
	defer b.statusMx.Unlock()

	newStatus := ping(ctx, params.Client, params.URL)

	// 0 -> 1
	if !b.status && newStatus {
		log.Print("[INFO] Broadcast started")
		b.status = true
		return time.Now()
	}

	// 1 -> 0
	// 0 -> 0
	if !newStatus {
		if b.status && lastOn.Add(params.DelayToOff).Before(time.Now()) {
			log.Print("[INFO] Broadcast finished")
			b.status = false
		}
		return lastOn
	}

	// 1 -> 1
	return time.Now()
}

// ping do get request to https://stream.radio-t.com and returns true on OK status and false for all other statuses
func ping(ctx context.Context, client http.Client, url string) (status bool) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		log.Printf("[WARN] unable to create %v request, %v", url, err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[DEBUG] unable to execute %v request, %v", url, err)
		return
	}
	defer resp.Body.Close() // nolint

	if resp.StatusCode == http.StatusOK {
		status = true
	}
	return
}

// nolint
func (b *BroadcastStatus) getStatus() bool {
	b.statusMx.Lock()
	defer b.statusMx.Unlock()
	return b.status
}

// ReactOn keys
func (b *BroadcastStatus) ReactOn() []string {
	return []string{}
}
