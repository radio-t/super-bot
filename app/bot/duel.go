//go:generate mockery -name TgRestrictingClient -case snake
package bot

import (
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/pkg/errors"
)

type duelMember struct {
	id       int
	userName string
}

type duel struct {
	requester, acceptor duelMember
	winner              *duelMember
	loser               *duelMember
	requestedAt         time.Time
}

func (d *duel) CalcRandWinner() {
	if rand.Int63()%2 == 0 {
		d.winner = &d.requester
		d.loser = &d.acceptor
	} else {
		d.winner = &d.acceptor
		d.loser = &d.requester
	}
}

type duelling struct {
	pending          map[string]duel
	recentRequesters map[int]time.Time
	mu               sync.Mutex

	acceptTimeout    time.Duration
	userDuelInterval time.Duration
}

func newDuelling(acceptTimeout, userDuelInterval time.Duration) *duelling {
	return &duelling{
		pending:          make(map[string]duel),
		recentRequesters: make(map[int]time.Time),
		acceptTimeout:    acceptTimeout,
		userDuelInterval: userDuelInterval,
	}
}

// backgroundCleanup:
// - cleaning old pending duels
// - handling user duel request limits
func (d *duelling) backgroundCleanup() {
	for range time.Tick(time.Second * 10) {
		d.mu.Lock()

		for requesterID, duel := range d.pending {
			if d.isDuelOutdated(duel.requestedAt) {
				delete(d.pending, requesterID)
			}
		}

		for requesterID, requestedAt := range d.recentRequesters {
			if time.Now().After(requestedAt.Add(d.userDuelInterval)) {
				delete(d.recentRequesters, requesterID)
			}
		}

		d.mu.Unlock()
	}
}

func (d *duelling) isDuelOutdated(requestedAt time.Time) bool {
	return time.Now().After(requestedAt.Add(d.acceptTimeout))
}

func (d *duelling) Begin(requesterUN string, requesterID int, acceptorUN string) error {
	// Without suicides please
	if requesterUN == acceptorUN {
		return errors.New("suicides is not allowed")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// User can't request new duel yet
	if _, ok := d.recentRequesters[requesterID]; ok {
		return errors.New("try again later")
	}

	d.recentRequesters[requesterID] = time.Now()
	d.pending[requesterUN] = duel{
		requester: duelMember{
			id:       requesterID,
			userName: requesterUN,
		},
		acceptor: duelMember{
			userName: acceptorUN,
		},
		requestedAt: time.Now(),
	}

	return nil
}

func (d *duelling) Accept(acceptorUN string, acceptorID int, requesterUN string) (duel, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pending, ok := d.pending[requesterUN]
	if !ok || pending.acceptor.userName != acceptorUN || d.isDuelOutdated(pending.requestedAt) {
		return duel{}, errors.New("there is no active duel")
	}

	pending.acceptor.id = acceptorID
	pending.CalcRandWinner()
	delete(d.pending, requesterUN)

	return pending, nil
}

type TgRestrictingClient interface {
	RestrictChatMember(config tbapi.RestrictChatMemberConfig) (tbapi.APIResponse, error)
}

// DuelBot providing duel bot
type DuelBot struct {
	duels *duelling
	tg    TgRestrictingClient
	su    SuperUser
}

func NewDuelBot(tgClient TgRestrictingClient, superuser SuperUser, acceptTimeout, userDuelInterval time.Duration) *DuelBot {
	duels := newDuelling(acceptTimeout, userDuelInterval)
	go duels.backgroundCleanup()
	return &DuelBot{duels: duels, tg: tgClient, su: superuser}
}

func (d *DuelBot) OnMessage(msg Message) (response Response) {
	duelReqMsg := duelRequestRe.FindString(msg.Text)
	duelAcceptRe := duelAcceptRe.FindString(msg.Text)

	switch {
	case duelReqMsg != "":
		// Tg doesn't provide any api to recognize if such user exists in chat :(
		// todo collect recently active users like banhammer bot?
		acceptorUsername := duelReqMsg[strings.Index(duelReqMsg, "@")+1:]

		if d.su.IsSuper(acceptorUsername) {
			return Response{Send: true, Text: fmt.Sprintf("@%s упал, замахиваясь перчаткой", msg.From.Username)}
		}

		err := d.duels.Begin(msg.From.Username, msg.From.ID, acceptorUsername)
		if err != nil {
			log.Printf("[DEBUG] creating duel (requestor %s, acceptor %s): %s", msg.From.Username, acceptorUsername, err.Error())
			return Response{}
		}

		return Response{
			Send: true,
			Text: fmt.Sprintf("*@%s вызвал на дуэль @%s!*", msg.From.Username, acceptorUsername),
		}
	case duelAcceptRe != "":
		reqUsername := duelAcceptRe[strings.Index(duelAcceptRe, "@")+1:]
		duel, err := d.duels.Accept(msg.From.Username, msg.From.ID, reqUsername)
		if err != nil {
			log.Printf("[DEBUG] accepting duel (acceptor %s, requestor %s): %s ", msg.From.Username, reqUsername, err.Error())
			return Response{}
		}

		err = d.mute(msg.ChatID, duel.loser.id)
		if err != nil {
			log.Printf("[ERROR] muting user %s (%d)", duel.loser.userName, duel.loser.id)
		} else {
			log.Printf("[DEBUG] user muted %s (%d)", duel.loser.userName, duel.loser.id)
		}

		return Response{
			Send: true,
			Text: fmt.Sprintf("*@%s* Победил! @%s прилег отдохнуть на сутки", duel.winner.userName, duel.loser.userName),
		}
	default:
		return Response{}
	}
}

var duelRequestRe = regexp.MustCompile(`(?m)[Бб]росить перчатку в @[a-zA-Z_0-9]{3,100}`)
var duelAcceptRe = regexp.MustCompile(`(?m)[Пп]оймать перчатку @[a-zA-Z_0-9]{3,100}`)

func (d *DuelBot) mute(chatID int64, userID int) error {
	resp, err := d.tg.RestrictChatMember(tbapi.RestrictChatMemberConfig{
		ChatMemberConfig: tbapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
		UntilDate:             time.Now().Add(time.Hour * 24).Unix(),
		CanSendMessages:       new(bool),
		CanSendMediaMessages:  new(bool),
		CanSendOtherMessages:  new(bool),
		CanAddWebPagePreviews: new(bool),
	})
	if err != nil {
		return errors.Wrapf(err, "restricting user")
	}

	if !resp.Ok {
		return errors.Wrapf(err, "restricting user: api response != ok")
	}

	return nil
}

func (d *DuelBot) ReactOn() []string {
	return []string{
		"(Б/б)росить перчатку в @username",
		"(П/п)оймать перчатку @username",
	}
}

// Help returns help message
func (d *DuelBot) Help() string {
	return genHelpMsg(d.ReactOn(), "предложить дуэль насмерть (mute на сутки)")
}
