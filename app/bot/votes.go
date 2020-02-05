package bot

import (
	"fmt"
	"strings"

	log "github.com/go-pkgz/lgr"
)

// Votes bot
type Votes struct {
	started bool
	votes   map[string]bool
	su      SuperUser
	topic   string
}

// NewVotes makes a bot for voting
func NewVotes(superUsers SuperUser) *Votes {
	return &Votes{su: superUsers}
}

// OnMessage pass msg to all bots and collects responses
func (v *Votes) OnMessage(msg Message) (response Response) {
	switch {
	case strings.HasPrefix(msg.Text, "++") && v.su.IsSuper(msg.From.Username):
		log.Printf("[INFO] voting started")
		v.votes = make(map[string]bool)
		v.started = true
		v.topic = msg.Text[2:]
		return Response{Text: fmt.Sprintf("голосование началось! (+1/-1) *%s*", v.topic), Send: true, Pin: false}
	case strings.HasPrefix(msg.Text, "!!") && v.su.IsSuper(msg.From.Username):
		log.Printf("[INFO] voting finished")
		v.started = false
		positiveNum, negativeNum := v.count(true), v.count(false)
		if (positiveNum + negativeNum) == 0 {
			positiveNum = 1
		}
		positivePerc := (100 * positiveNum) / (positiveNum + negativeNum)
		negativePerc := (100 * negativeNum) / (positiveNum + negativeNum)
		return Response{
			Text: fmt.Sprintf("голосование завершено - _%s_\n- *за: %d%% (%d)*\n- *против: %d%% (%d) *",
				v.topic, positivePerc, positiveNum, negativePerc, negativeNum),
			Send: true,
		}
	case (msg.Text == "+1" || strings.Contains(msg.Text, ":+1:")) && v.started:
		if _, found := v.votes[msg.From.Username]; !found {
			v.votes[msg.From.Username] = true
			log.Printf("[DEBUG] vote +1 from %s", msg.From.DisplayName)
		}
	case (msg.Text == "-1" || strings.Contains(msg.Text, ":-1:")) && v.started:
		if _, found := v.votes[msg.From.Username]; !found {
			v.votes[msg.From.Username] = false
			log.Printf("[DEBUG] vote +1 from %s", msg.From.DisplayName)
		}
	}
	return Response{}
}

// ReactOn keys
func (v Votes) ReactOn() []string {
	return []string{}
}

func (v Votes) count(side bool) int {
	res := 0
	for _, b := range v.votes {
		if b == side {
			res++
		}
	}
	return res
}
