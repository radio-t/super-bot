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
	luckFactor  float64
}

// NewWTF makes a random ban bot
func NewWTF(minDuration, maxDuration time.Duration, luckFactor float64) *WTF {
	log.Printf("[INFO] WTF bot with %v-%v interval, lucky %.2f", minDuration, maxDuration, luckFactor)
	rand.Seed(time.Now().UnixNano())
	return &WTF{minDuration: minDuration, maxDuration: maxDuration, luckFactor: luckFactor}
}

// OnMessage sets duration of ban randomly
func (w WTF) OnMessage(msg Message) (response Response) {

	if !contains(w.ReactOn(), msg.Text) {
		return Response{}
	}

	if rand.Float64() < w.luckFactor {
		return Response{Text: fmt.Sprintf("@%s, на этот раз тебе повезло!", msg.From.Username), Send: true}
	}

	banDuration := w.minDuration + time.Second*time.Duration(rand.Int63n(int64(w.maxDuration.Seconds()-w.minDuration.Seconds())))
	return Response{
		Text:        fmt.Sprintf("@%s получает бан на %v", msg.From.Username, banDuration),
		Send:        true,
		BanInterval: banDuration,
	}
}

// ReactOn keys
func (w WTF) ReactOn() []string {
	return []string{"wtf!"}
}

// Help returns help message
func (w WTF) Help() string {
	return genHelpMsg(w.ReactOn(), "если не повезет, блокирует пользователя на какое-то время")
}
