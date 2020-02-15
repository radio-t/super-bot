package events

import (
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/radio-t/super-bot/app/bot"
)

// Terminator helps to block too active users
type Terminator struct {
	BanDuration   time.Duration
	BanPenalty    int
	AllowedPeriod time.Duration
	Exclude       SuperUser
	users         map[bot.User]activity
}

type activity struct {
	lastActivity time.Time
	penalty      int
}

type ban struct {
	active bool
	new    bool
}

// check if user bothered bot too often and ban for BanDuration
func (t *Terminator) check(user bot.User, sent time.Time) ban {

	noBan := ban{active: false, new: false}
	if t.Exclude.IsSuper(user.Username) {
		return noBan
	}

	if t.users == nil {
		t.users = make(map[bot.User]activity)
		log.Printf("[DEBUG] terminator with BanDuration=%v, BanPenalty=%d, excluded=%v", t.BanDuration, t.BanPenalty, t.Exclude)
	}

	info, found := t.users[user]
	if !found {
		t.users[user] = activity{lastActivity: sent}
		return noBan
	}

	if time.Now().Before(info.lastActivity.Add(t.AllowedPeriod)) {
		info.penalty++
		log.Printf("[DEBUG] penalty increased for %v to %d", user, info.penalty)
	} else {
		if info.penalty > 0 {
			log.Printf("[DEBUG] penalty reset for %v from %d", user, info.penalty)
		}
		info.penalty = 0
	}

	if info.penalty == t.BanPenalty {
		log.Printf("[WARN] banned %v", user)
		info.penalty++
		t.users[user] = info
		return ban{active: true, new: true}
	}

	if info.penalty >= t.BanPenalty {
		log.Printf("[DEBUG] still banned %v", user)
		return ban{active: true, new: false}
	}

	info.lastActivity = sent
	t.users[user] = info
	return noBan
}
