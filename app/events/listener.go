package events

import (
	"fmt"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/sromku/go-gitter"

	"git.umputun.com/radio-t/gitter-rt-bot/app/bot"
	"git.umputun.com/radio-t/gitter-rt-bot/app/reporter"
)

// Listener of gitter events
type Listener struct {
	Terminator
	reporter.Reporter

	AutoBan AutoBan
	API     *gitter.Gitter
	RoomID  string
	Bots    bot.Interface
}

// Do process all events, blocked call
func (l Listener) Do() {
	log.Printf("[INFO] activated for room=%s", l.RoomID)
	// l.API.SetDebug(true, nil)
	// l.API.SendMessage(l.RoomID, "**joined and activated**")

	go l.killAfter(time.Hour * 24)

	stream := l.API.Stream(l.RoomID)
	go l.API.Listen(stream)

	for {
		event := <-stream.Event

		switch ev := event.Data.(type) {

		case *gitter.MessageReceived:
			log.Printf(" -> %s: %s: [%s]",
				ev.Message.From.DisplayName, ev.Message.Sent.Format("2006-01-02 15:04:05"), ev.Message.Text)

			if l.AutoBan.check(ev.Message) {
				continue
			}

			l.Save(ev.Message)

			if ev.Message.From.DisplayName == "радио-т бот" {
				log.Printf("[DEBUG] ignore %+v", ev.Message)
				continue
			}

			if resp, send := l.Bots.OnMessage(ev.Message); send {
				log.Printf("[DEBUG] bot sent - %+v", resp)
				if ban := l.check(ev.Message.From); ban.active {
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
			log.Printf("[FATAL] connection closed")
		}
	}
}

// killAfter will close the app after some period. gitter streams doesn't like long sessions.
// the app expected to be restarted by external watchers, like docker's restart policy.
func (l Listener) killAfter(killDuration time.Duration) {
	timer := time.NewTimer(killDuration)
	<-timer.C
	log.Printf("[FATAL] internal kill initiated")
}
