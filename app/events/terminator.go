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
func (t *Terminator) check(user bot.User) ban {

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
		t.users[user] = activity{lastActivity: time.Now()}
		return noBan
	}

	if info.penalty >= t.BanPenalty {
		if time.Now().Before(info.lastActivity.Add(t.BanDuration)) {
			if info.penalty == t.BanPenalty {
				log.Printf("[WARN] banned %s", user)
				info.penalty++
				t.users[user] = info
				return ban{active: true, new: true}
			}
			log.Printf("[DEBUG] still banned %s", user)
			return ban{active: true, new: false}
		}
		info.penalty = 0
		log.Printf("[INFO] unbanned %s", user)
	}

	if time.Now().Before(info.lastActivity.Add(t.AllowedPeriod)) {
		info.penalty++
		log.Printf("[DEBUG] penalty increased for %s to %d", user, info.penalty)
	} else {
		if info.penalty > 0 {
			log.Printf("[DEBUG] penalty reset for %s from %d", user, info.penalty)
		}
		info.penalty = 0
	}

	info.lastActivity = time.Now()
	t.users[user] = info
	return noBan
}
