package events

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	MsgBroadcastStarted  = "вещание началось"
	MsgBroadcastFinished = "вещание завершилось"
)

// BroadcastStatusBot responsible for tracking broadcast status and notify users about status change
// I don't place it in bot package because it's more like a rtjc rather than other bots
type BroadcastStatusBot struct {
	pingInterval    time.Duration // Ping interval
	broadcastUrl    string        // Url for "ping"
	delayToOff      time.Duration // Interval of not o status to swithc to OFF
	persistFileName string        // If set, status will be persisted to file and loaded on start
	// It can help in case of service restart during broadcasting
	status          bool      // Current broadcast status true - ON, false - OFF
	lastOnStateTime time.Time // Last time broadcast was in ON state
	submitter       Submitter
	pingFn          func(ctx context.Context) bool
}

func NewBroadcastStatusBot(pingInterval time.Duration, broadcastUrl string, delayToOff time.Duration, persistFileName string, submitter Submitter) *BroadcastStatusBot {
	bot := &BroadcastStatusBot{pingInterval: pingInterval, broadcastUrl: broadcastUrl, delayToOff: delayToOff, persistFileName: persistFileName, status: false, submitter: submitter}
	bot.pingFn = bot.ping
	return bot
}

func (b BroadcastStatusBot) Start(ctx context.Context) {
	b.loadStatus()
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(b.pingInterval):
			b.process(ctx)
		}
	}
}

func (b *BroadcastStatusBot) process(ctx context.Context) {
	newStatus := b.pingFn(ctx)
	if newStatus {
		b.lastOnStateTime = time.Now()
	}
	if b.status == newStatus {
		return
	}

	if !b.status && newStatus { // off -> on
		b.statusChanged(ctx, true)
		return
	}

	if b.status && !newStatus { // on -> off
		if time.Now().After(b.lastOnStateTime.Add(b.delayToOff)) {
			b.statusChanged(ctx, false)
			return
		}

		log.Printf("[DEBUG] Ping not success. %v to change status to OFF", b.lastOnStateTime.Add(b.delayToOff).Sub(time.Now()))
	}

}

func (b BroadcastStatusBot) statusChanged(ctx context.Context, newStatus bool) {
	log.Printf("[INFO] Broadcast status changed: %v -> %v", b.status, newStatus)
	msg := MsgBroadcastFinished
	if newStatus {
		msg = MsgBroadcastStarted
	}
	b.saveStatus(newStatus)
	b.submitter.Submit(ctx, msg)
}

// ping do get request to https://stream.radio-t.com and returns true on OK status and false for all other statuses
func (b BroadcastStatusBot) ping(ctx context.Context) bool {
	req, err := http.NewRequest("GET", b.broadcastUrl, nil)
	if err != nil {
		log.Printf("[WARN] unable to created icecast request, %v", err)
		return false
	}

	req = req.WithContext(ctx)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[WARN] unable to do %v request, %v", b.broadcastUrl, err)
		return false
	}
	defer resp.Body.Close()

	log.Printf("[DEBUG] %v : %v status", b.broadcastUrl, resp.Status)
	return resp.StatusCode == http.StatusOK
}

// loadStatus from file in case of restart
func (b *BroadcastStatusBot) loadStatus() {
	b.status = false // by default status=OFF
	if b.persistFileName == "" {
		return
	}

	data, err := ioutil.ReadFile(b.persistFileName)
	if err != nil {
		log.Printf("[WARN] unable to load status, %v", err)
		return
	}

	var status bool
	if err := json.Unmarshal(data, &status); err != nil {
		log.Printf("[WARN] unable to unmarshal status %s, %v", data, err)
		return
	}

	b.status = status
}

// saveStatus in case of change
func (b BroadcastStatusBot) saveStatus(newStatus bool) {
	b.status = newStatus

	if b.persistFileName == "" {
		return
	}

	data, err := json.Marshal(newStatus)
	if err != nil {
		log.Printf("[WARN] unable to marshal status %v, %v", newStatus, err)
		return
	}

	if err := ioutil.WriteFile(b.persistFileName, data, os.ModePerm); err != nil {
		log.Printf("[WARN] unable to save status %v, %v", b.status, err)
	}
}
