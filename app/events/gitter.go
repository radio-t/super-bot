package events

import (
	"context"
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/sromku/go-gitter"

	"github.com/radio-t/gitter-rt-bot/app/bot"
	"github.com/radio-t/gitter-rt-bot/app/reporter"
)

// GitterListener of gitter events
type GitterListener struct {
	Terminator
	reporter.Reporter

	API    *gitter.Gitter
	RoomID string
	Bots   bot.Interface
}

// Do process all events, blocked call
func (l *GitterListener) Do(ctx context.Context) {
	log.Printf("[INFO] activated for room=%s", l.RoomID)
	// l.API.SetDebug(true, nil)
	// l.API.SendMessage(l.RoomID, "**joined and activated**")

	go l.killAfter(time.Hour * 24)

	stream := l.API.Stream(l.RoomID)
	go l.API.Listen(stream)

	for {

		var event gitter.Event

		select {
		case event = <-stream.Event:
		case <-ctx.Done():
			return
		}

		switch ev := event.Data.(type) {

		case *gitter.MessageReceived:
			log.Printf(" -> %s: %s: [%s]",
				ev.Message.From.DisplayName, ev.Message.Sent.Format("2006-01-02 15:04:05"), ev.Message.Text)

			m := bot.Message{
				Text: ev.Message.Text,
				HTML: ev.Message.HTML,
				Sent: ev.Message.Sent,
				From: bot.User{
					ID:          ev.Message.From.ID,
					Username:    ev.Message.From.Username,
					DisplayName: ev.Message.From.DisplayName,
				},
			}
			l.Save(m)

			if ev.Message.From.DisplayName == "радио-т бот" {
				log.Printf("[DEBUG] ignore %+v", ev.Message)
				continue
			}

			if resp, send := l.Bots.OnMessage(m); send {
				log.Printf("[DEBUG] bot sent - %+v", resp)
				if ban := l.check(m.From); ban.active {
					if ban.new {
						m := fmt.Sprintf("@%s _тебя слишком много, отдохни ..._", ev.Message.From.Username)
						if _, err := l.API.SendMessage(l.RoomID, m); err != nil {
							log.Printf("[WARN] failed to send, %v", err)
						}
					}
				} else {
					if _, err := l.API.SendMessage(l.RoomID, resp); err != nil {
						log.Printf("[WARN] failed to send, %v", err)
					}
				}
			}
		case *gitter.GitterConnectionClosed:
			log.Fatalf("[ERROR] connection closed")
		}
	}
}

// killAfter will close the app after some period. gitter streams doesn't like long sessions.
// the app expected to be restarted by external watchers, like docker's restart policy.
func (l *GitterListener) killAfter(killDuration time.Duration) {
	timer := time.NewTimer(killDuration)
	<-timer.C
	log.Fatalf("[WARN] internal kill initiated")
}
