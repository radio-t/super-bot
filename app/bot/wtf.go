package bot

import (
	"fmt"
	"math/rand"
	"time"

	log "github.com/go-pkgz/lgr"
)

// WTF bot bans user for random interval
type WTF struct {
	minDuration time.Duration
	maxDuration time.Duration
	rand        func(n int64) int64
}

// NewWTF makes a random ban bot
func NewWTF(minDuration, maxDuration time.Duration) *WTF {
	log.Printf("[INFO] WTF bot with %v-%v interval", minDuration, maxDuration)
	rand.Seed(time.Now().UnixNano())
	return &WTF{minDuration: minDuration, maxDuration: maxDuration, rand: rand.Int63n}
}

// OnMessage sets duration of ban randomly
func (w WTF) OnMessage(msg Message) (response Response) {
	if !contains(w.ReactOn(), msg.Text) {
		return Response{}
	}

	mention := "@" + msg.From.Username
	if msg.From.Username == "" {
		mention = msg.From.DisplayName
	}

	banDuration := w.minDuration + time.Second*time.Duration(w.rand(int64(w.maxDuration.Seconds()-w.minDuration.Seconds())))
	return Response{
		Text:        fmt.Sprintf("[%s](tg://user?id=%d) получает бан на %v", mention, msg.From.ID, HumanizeDuration(banDuration)),
		Send:        true,
		BanInterval: banDuration,
	}
}

// ReactOn keys
func (w WTF) ReactOn() []string {
	return []string{"wtf!", "wtf?"}
}

// Help returns help message
func (w WTF) Help() string {
	return genHelpMsg(w.ReactOn(), "если не повезет, блокирует пользователя на какое-то время")
}
