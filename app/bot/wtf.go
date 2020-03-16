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
}

// NewWTF makes a random ban bot
func NewWTF(minDuration, maxDuration time.Duration) *WTF {
	log.Printf("[INFO] WTF bot with %v-%v interval", minDuration, maxDuration)
	rand.Seed(time.Now().UnixNano())
	return &WTF{minDuration: minDuration, maxDuration: maxDuration}
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

	banDuration := w.minDuration + time.Second*time.Duration(rand.Int63n(int64(w.maxDuration.Seconds()-w.minDuration.Seconds())))
	return Response{
		Text:        fmt.Sprintf("[%s](tg://user?id=%d) получает бан на %s", mention, msg.From.ID, getHumanDuration(banDuration)),
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

func getHumanDuration(d time.Duration) string {

	var str string

	seconds := int64(d / time.Second)

	days := int64(seconds / (24 * 60 * 60))
	if days > 0 {
		str = fmt.Sprintf("%dd", days)
		seconds = seconds - days*(24*60*60)
	}

	hours := int64(seconds / (60 * 60))
	if hours > 0 || len(str) > 0 {
		str = fmt.Sprintf("%s%dh", str, hours)
		seconds = seconds - hours*(60*60)
	}

	minutes := int64(seconds / 60)
	if minutes > 0 || len(str) > 0 {
		str = fmt.Sprintf("%s%dm", str, minutes)
		seconds = seconds - minutes*60
	}

	str = fmt.Sprintf("%s%ds", str, seconds)

	return str
}
