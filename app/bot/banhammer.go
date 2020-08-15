package bot

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

//go:generate mockery -name TgBanClient -case snake

// Banhammer bot, allows (super users only) to ban or unban anyone
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
	KickChatMember(config tbapi.KickChatMemberConfig) (tbapi.APIResponse, error)
	UnbanChatMember(config tbapi.ChatMemberConfig) (tbapi.APIResponse, error)
}

// NewBanhammer makes a bot for admins reacting on ban!user unban!user
func NewBanhammer(tgClient TgBanClient, superUser SuperUser, maxRecentUsers int) *Banhammer {
	log.Printf("[INFO] Banhammer bot, max users to keep: %d, supers: %v", maxRecentUsers, superUser)
	return &Banhammer{tgClient: tgClient, superUser: superUser, recentUsers: map[string]userInfo{}, maxRecentUsers: maxRecentUsers}
}

// Help returns help message
func (b *Banhammer) Help() string {
	return genHelpMsg(b.ReactOn(), "забанить/разбанить (только для админов)")
}

// ReactOn keys
func (b *Banhammer) ReactOn() []string {
	return []string{"ban!", "unban!"}
}

// OnMessage pass msg to all bots and collects responses
// In order to translate user name to ID (mandatory for tg kick/unban) collect up to maxRecentUsers recently seen users
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
		_, err := b.tgClient.KickChatMember(tbapi.KickChatMemberConfig{
			ChatMemberConfig: tbapi.ChatMemberConfig{UserID: user.ID, ChatID: msg.ChatID},
		})
		if err != nil {
			log.Printf("[WARN] failed to ban %s, %v", name, err)
			return Response{}
		}
		log.Printf("[INFO] banned %+v", user.User)
		return Response{Text: fmt.Sprintf("прощай %s", name), Send: true}
	case "unban":
		_, err := b.tgClient.UnbanChatMember(tbapi.ChatMemberConfig{UserID: user.ID, ChatID: msg.ChatID})
		if err != nil {
			log.Printf("[WARN] failed to unban %s, %v", name, err)
			return Response{}
		}
		log.Printf("[INFO] unbanned %+v", user.User)
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
	// remove 10% of oldest records
	for i := 0; i < b.maxRecentUsers/10; i++ {
		delete(b.recentUsers, users[i].Username)
	}
}

func (b *Banhammer) parse(text string) (react bool, cmd, name string) {

	for _, prefix := range b.ReactOn() {
		if strings.HasPrefix(text, prefix) {
			return true, strings.TrimSuffix(prefix, "!"),
				strings.Replace(strings.TrimSpace(strings.TrimPrefix(text, prefix)), " ", "+", -1)
		}
	}
	return false, "", ""
}
