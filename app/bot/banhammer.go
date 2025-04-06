package bot

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

//go:generate moq --out mocks/tg_ban_client.go --pkg mocks --skip-ensure . TgBanClient:TgBanClient

// Banhammer bot, allows (superusers only) to ban or unban anyone
type Banhammer struct {
	tgClient  TgBanClient
	superUser SuperUser

	maxRecentUsers int
	recentUsers    map[string]userInfo
}

type userInfo struct {
	User
	ts time.Time
}

// TgBanClient is a subset of tg api limited to ban-related operations only
type TgBanClient interface {
	Request(c tbapi.Chattable) (*tbapi.APIResponse, error)
}

// NewBanhammer makes a bot for admins reacting on ban!user unban!user
func NewBanhammer(tgClient TgBanClient, superUser SuperUser, maxRecentUsers int) *Banhammer {
	log.Printf("[INFO] Banhammer bot, max users to keep: %d, supers: %v", maxRecentUsers, superUser)
	return &Banhammer{tgClient: tgClient, superUser: superUser, recentUsers: map[string]userInfo{}, maxRecentUsers: maxRecentUsers}
}

// Help returns help message
func (b *Banhammer) Help() string {
	return GenHelpMsg(b.ReactOn(), "забанить/разбанить (только для админов)")
}

// ReactOn keys
func (b *Banhammer) ReactOn() []string {
	return []string{"ban!", "unban!"}
}

// OnMessage pass msg to all bots and collects responses
// In order to translate username to ID (mandatory for tg kick/unban) collect up to maxRecentUsers recently seen users
func (b *Banhammer) OnMessage(msg Message) (response Response) {

	// update list of recent users
	b.recentUsers[msg.From.Username] = userInfo{User: msg.From, ts: time.Now()}
	if len(b.recentUsers) > b.maxRecentUsers {
		b.cleanup()
	}

	ok, cmd, name := b.parse(msg.Text)
	if !ok || !b.superUser.IsSuper(msg.From.Username) { // only super may ban/unban
		return Response{}
	}

	if b.superUser.IsSuper(strings.TrimPrefix(name, "@")) { // super can't be banned by another super
		return Response{}
	}

	user, found := b.recentUsers[strings.TrimPrefix(name, "@")]
	if !found {
		log.Printf("[WARN] can't get ID for user %s", name)
		return Response{}
	}

	switch cmd {
	case "ban":
		_, err := b.tgClient.Request(tbapi.KickChatMemberConfig{
			ChatMemberConfig: tbapi.ChatMemberConfig{UserID: user.ID, ChatID: msg.ChatID},
		})
		if err != nil {
			log.Printf("[WARN] failed to ban %s, %v", name, err)
			return Response{}
		}
		log.Printf("[INFO] banned %+v by %+v", user.User, msg.From)
		return Response{Text: fmt.Sprintf("прощай %s", name), Send: true}
	case "unban":
		_, err := b.tgClient.Request(tbapi.UnbanChatMemberConfig{ChatMemberConfig: tbapi.ChatMemberConfig{UserID: user.ID, ChatID: msg.ChatID}})
		if err != nil {
			log.Printf("[WARN] failed to unban %s, %v", name, err)
			return Response{}
		}
		log.Printf("[INFO] unbanned %+v by %+v", user.User, msg.From)
		return Response{Text: fmt.Sprintf("амнистия для %s", name), Send: true}
	}

	return Response{}
}

func (b *Banhammer) cleanup() {
	users := make([]userInfo, len(b.recentUsers))
	for _, u := range b.recentUsers {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].ts.Before(users[j].ts) })
	// remove 10% of the oldest records
	for i := 0; i < b.maxRecentUsers/10; i++ {
		delete(b.recentUsers, users[i].Username)
	}
}

func (b *Banhammer) parse(text string) (react bool, cmd, name string) {

	for _, prefix := range b.ReactOn() {
		if strings.HasPrefix(text, prefix) {
			return true, strings.TrimSuffix(prefix, "!"),
				strings.ReplaceAll(strings.TrimSpace(strings.TrimPrefix(text, prefix)), " ", "+")
		}
	}
	return false, "", ""
}
