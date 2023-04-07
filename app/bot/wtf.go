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
	return &WTF{minDuration: minDuration, maxDuration: maxDuration, rand: rand.Int63n, superUser: superUser, tooSoon: time.Minute}
}

// OnMessage sets duration of ban randomly
func (w *WTF) OnMessage(msg Message) (response Response) {

	wtfContains := WTFSteroidChecker{
		Message: msg.Text}

	if !wtfContains.Contains() {
		return Response{}
	}

	wtfUser := msg.From
	var wtfChannelID int64
	var wtfChannelUsername string
	if w.superUser.IsSuper(msg.From.Username) {
		if msg.ReplyTo.From.ID == 0 { // not reply, ignore for supers
			return Response{}
		}
		wtfRequested := fmt.Sprintf("[INFO] wtf requested by %q for %q, id:%d", msg.From.Username, msg.ReplyTo.From.Username, msg.ReplyTo.From.ID)
		if msg.ReplyTo.SenderChat.ID != 0 {
			wtfChannelID = msg.ReplyTo.SenderChat.ID
			wtfChannelUsername = msg.ReplyTo.SenderChat.UserName
			wtfRequested = fmt.Sprintf("[INFO] wtf requested by %q for %q, id:%d", msg.From.Username, wtfChannelUsername, wtfChannelID)
		}
		log.Print(wtfRequested)
		wtfUser = msg.ReplyTo.From // set WTF user from ReplyTo.From for supers, so it will ban the user replied to
	}

	if w.superUser.IsSuper(wtfUser.Username) {
		log.Printf("[WARN] WTF requested of user %q, ignored (super)", wtfUser.Username)
		return Response{} // don't allow supers to ban other supers
	}

	// message from channel, not banned by superuser above
	if msg.From.ID == 136817688 && wtfChannelID == 0 {
		wtfChannelID = msg.SenderChat.ID
		wtfChannelUsername = msg.SenderChat.UserName
	}

	mention := "@" + wtfUser.Username
	if wtfUser.Username == "" {
		mention = wtfUser.DisplayName
	}
	if wtfChannelID != 0 {
		mention = "@" + wtfChannelUsername
	}

	banDuration := w.minDuration + time.Second*time.Duration(w.rand(int64(w.maxDuration.Seconds()-w.minDuration.Seconds())))
	if time.Since(w.lastWtf) < w.tooSoon { // if last wtf was less than 1 minute ago, add more time to ban duration
		increase := time.Hour * 5 * time.Duration(w.tooSoon.Seconds()-time.Since(w.lastWtf).Seconds())
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

	durationString := HumanizeDuration(banDuration)
	if wtfChannelID != 0 {
		durationString = "навсегда"
	}

	return Response{
		Text:        fmt.Sprintf("%s получает бан на %v", EscapeMarkDownV1Text(mention), durationString),
		Send:        true,
		BanInterval: banDuration,
		User:        wtfUser,
		ChannelID:   wtfChannelID,
	}
}

// ReactOn keys
func (w *WTF) ReactOn() []string {
	return []string{"wtf!", "wtf?"}
}

// Help returns help message
func (w *WTF) Help() string {
	return GenHelpMsg(w.ReactOn(), "если не повезет, блокирует пользователя на какое-то время")
}
