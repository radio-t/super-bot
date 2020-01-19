package bot

import (
	"context"
	"net/http"
	"time"

	log "github.com/go-pkgz/lgr"
)

const (
	MsgBroadcastStarted  = "Вещание началось"
	MsgBroadcastFinished = "Вещание завершилось"
)

// BroadcastStatus bot returns current broadcast status
type BroadcastStatus struct {
	// Params
	broadcastUrl string        // Url for "ping"
	pingInterval time.Duration // Ping interval
	delayToOff   time.Duration // After this interval of not OK, status will be switcher to OFF

	// State
	status         bool  // current broadcast status
	lastSentStatus *bool // by default this value is null to react on first message
	offPeriod      time.Duration
	lastCheck      time.Time

	// ping func (for tests)
	ping func(ctx context.Context, url string) bool
}

// NewBroadcastStatus starts status checking goroutine and returns bot instance
func NewBroadcastStatus(ctx context.Context, broadcastUrl string, pingInterval time.Duration, delayToOff time.Duration) *BroadcastStatus {
	log.Printf("[INFO] BroadcastStatus bot with %v", broadcastUrl)
	b := &BroadcastStatus{broadcastUrl: broadcastUrl, pingInterval: pingInterval, delayToOff: delayToOff, lastCheck: time.Now(), ping: ping}
	go b.checker(ctx)
	return b
}

// OnMessage returns current broadcast status
func (b *BroadcastStatus) OnMessage(_ Message) (response string, answer bool) {
	if b.lastSentStatus != nil && b.status == *b.lastSentStatus {
		return
	}

	answer = true
	response = MsgBroadcastStarted

	status := b.status
	b.lastSentStatus = &status
	if !b.status {
		response = MsgBroadcastFinished
	}
	return
}

func (b *BroadcastStatus) checker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(b.pingInterval):
			b.check(ctx)
		}
	}
}

func (b *BroadcastStatus) check(ctx context.Context) {
	defer func() {
		b.lastCheck = time.Now()
	}()

	newStatus := b.ping(ctx, b.broadcastUrl)

	if !b.status && newStatus {
		log.Print("[INFO] Broadcast started")
		b.status = true
		b.offPeriod = 0
		return
	}

	if b.status && !newStatus {
		b.offPeriod += time.Now().Sub(b.lastCheck)
		if b.offPeriod > b.delayToOff {
			log.Print("[INFO] Broadcast finished")
			b.status = false
			b.offPeriod = 0
			return
		}
	}
}

// ping do get request to https://stream.radio-t.com and returns true on OK status and false for all other statuses
func ping(ctx context.Context, url string) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("[WARN] unable to created %v request, %v", url, err)
		return false
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		log.Printf("[WARN] unable to do %v request, %v", url, err)
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// ReactOn keys
func (b *BroadcastStatus) ReactOn() []string {
	return []string{}
}
