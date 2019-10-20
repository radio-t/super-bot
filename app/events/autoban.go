package events

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/sromku/go-gitter"
)

type banActivity struct {
	dt    time.Time
	count int
	dups  int
	msg   string
}

// AutoBan reacts on unusually long messages and too-many-per-sec kind of activity.
// It doesn't use gitter module (ban not supported by go-gitter) but sends ban POST command directly to API
type AutoBan struct {
	GitterToken  string
	RoomID       string
	SuperUsers   SuperUser
	MaxMsgSize   int
	MsgsPerSec   int
	DupsPerSec   int
	lastActivity map[gitter.User]banActivity
}

// check activity for bad patterns. not thread-safe
func (a *AutoBan) check(msg gitter.Message) bool {

	if a.lastActivity == nil {
		a.lastActivity = make(map[gitter.User]banActivity)
	}

	if a.SuperUsers.IsSuper(msg.From) || msg.From.DisplayName == "радио-т бот" {
		return false
	}

	if len(msg.Text) > a.MaxMsgSize {
		log.Printf("[INFO] triggered ban due size, %v", msg.From)
		a.ban(msg.From)
		return true
	}

	if act, found := a.lastActivity[msg.From]; found {

		if act.msg == msg.Text {
			act.dups++
			if a.DupsPerSec != 0 && act.dups >= a.DupsPerSec {
				log.Printf("[INFO] triggered ban due same second dups, %v", msg.From)
				a.ban(msg.From)
				return true
			}
			a.lastActivity[msg.From] = act
			log.Printf("[DEBUG] same-second dup from %s, count=%d", msg.From.Username, act.dups)
			return false
		}

		if act.count >= a.MsgsPerSec {
			log.Printf("[WARN] triggered ban due activity, %v", msg.From)
			a.ban(msg.From)
			return true
		}

		if time.Now().Before(act.dt.Add(time.Second)) {
			act.dt = time.Now()
			act.count++
			act.msg = msg.Text
			a.lastActivity[msg.From] = act
			log.Printf("[DEBUG] same-second message from %s, count=%d", msg.From.Username, act.count)
			return false
		}
	}
	a.lastActivity[msg.From] = banActivity{dt: time.Now(), count: 1, dups: 1, msg: msg.Text}
	return false
}

// ban API undocumented, based on https://github.com/gitterHQ/gitter/issues/989#issuecomment-163594444
func (a *AutoBan) ban(user gitter.User) {
	log.Printf("[INFO] autoban for %s/%s", user.Username, user.DisplayName)

	banUser := []byte(fmt.Sprintf(`{"username": "%s"}`, user.Username))
	req, err := http.NewRequest("POST", fmt.Sprintf("https://gitter.im/api/v1/rooms/%s/bans?access_token=%s",
		a.RoomID, a.GitterToken), bytes.NewReader(banUser))
	if err != nil {
		log.Printf("[WARN] failed to make request to ban %s, error=%v", user.Username, err)
		return
	}
	client := http.Client{Timeout: time.Second * 5}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.GitterToken))
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send ban request, %v", err)
		return
	}
	log.Printf("[INFO] ban status for %s - %s", user.Username, resp.Status)
	_ = resp.Body.Close()
}
