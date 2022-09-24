package bot

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

// WTF bot bans user for random interval
type WTF struct {
	superUser   SuperUser
	minDuration time.Duration
	maxDuration time.Duration
	rand        func(n int64) int64 // tests may change it
	lastWtf     time.Time
	tooSoon     time.Duration // tests may change it
}

// NewWTF makes a random ban bot
func NewWTF(minDuration, maxDuration time.Duration, superUser SuperUser) *WTF {
	log.Printf("[INFO] WTF bot with %v-%v interval", minDuration, maxDuration)
	rand.Seed(time.Now().UnixNano())
	return &WTF{minDuration: minDuration, maxDuration: maxDuration, rand: rand.Int63n, superUser: superUser, tooSoon: time.Minute}
}

// OnMessage sets duration of ban randomly
func (w *WTF) OnMessage(msg Message) (response Response) {

	wtfContains := WTFSteroidChecker{
		message: msg.Text}

	if !wtfContains.Contains() {
		return Response{}
	}

	if w.superUser.IsSuper(msg.From.Username) {
		if msg.ReplyTo.From.ID == 0 { // not reply, ignore for supers
			return Response{}
		}
		log.Printf("[INFO] wtf requested by %q for %q:%d", msg.From.Username, msg.ReplyTo.From.Username, msg.ReplyTo.From.ID)
		msg.From = msg.ReplyTo.From // set From from ReplyTo.From for supers, so it will ban the user replied to
	}

	if w.superUser.IsSuper(msg.From.Username) {
		return Response{} // don't allow supers to ban other supers
	}

	mention := "@" + msg.From.Username
	if msg.From.Username == "" {
		mention = msg.From.DisplayName
	}

	banDuration := w.minDuration + time.Second*time.Duration(w.rand(int64(w.maxDuration.Seconds()-w.minDuration.Seconds())))
	if time.Since(w.lastWtf) < w.tooSoon { // if last wtf was less than 1 minute ago, add more to result
		increase := time.Hour * time.Duration(w.tooSoon.Seconds()-time.Since(w.lastWtf).Seconds())
		log.Printf("[INFO] wtf from %v in %v is too soon, adding %v to ban duration", mention, time.Since(w.lastWtf), increase)
		banDuration += increase
	}
	switch w.rand(10) {
	case 1:
		banDuration = time.Hour * 666
	case 2:
		banDuration = time.Minute*77 + time.Second*7
	}
	w.lastWtf = time.Now()

	return Response{
		Text:        fmt.Sprintf("[%s](tg://user?id=%d) получает бан на %v", mention, msg.From.ID, HumanizeDuration(banDuration)),
		Send:        true,
		BanInterval: banDuration,
		User:        msg.From,
	}
}

// ReactOn keys
func (w *WTF) ReactOn() []string {
	return []string{"wtf!", "wtf?"}
}

// Help returns help message
func (w *WTF) Help() string {
	return genHelpMsg(w.ReactOn(), "если не повезет, блокирует пользователя на какое-то время")
}
