package bot

import (
	"fmt"
	"log"
	"strings"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

//go:generate mockery -name TgBanClient -case snake

// Banhammer bot, allows (super users only) to ban or unban anyone
type Banhammer struct {
	tgClient  TgBanClient
	superUser SuperUser
}

// TgBanClient is a subset of tg api limited to ban-related operations only
type TgBanClient interface {
	KickChatMember(config tbapi.KickChatMemberConfig) (tbapi.APIResponse, error)
	UnbanChatMember(config tbapi.ChatMemberConfig) (tbapi.APIResponse, error)
}

// NewBanhammer makes a bot for admins reacting on ban!user unban!user
func NewBanhammer(tgClient TgBanClient, superUser SuperUser) *Banhammer {
	log.Printf("[INFO] Banhammer bot")
	return &Banhammer{tgClient: tgClient, superUser: superUser}
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
func (b *Banhammer) OnMessage(msg Message) (response Response) {

	ok, cmd, name := b.parse(msg.Text)
	if !ok || !b.superUser.IsSuper(msg.From.Username) {
		return Response{}
	}

	userID := getUserID(name, msg.Entities)

	userCfg := tbapi.ChatMemberConfig{
		ChatID:          msg.ChatID,
		ChannelUsername: name,
		UserID:          userID,
	}

	switch cmd {
	case "ban":

		_, err := b.tgClient.KickChatMember(tbapi.KickChatMemberConfig{ChatMemberConfig: userCfg})
		if err != nil {
			log.Printf("[WARN] failed to ban %s, %v", name, err)
			return Response{}
		}
		return Response{Text: fmt.Sprintf("прощай %s", name), Send: true}
	case "unban":
		_, err := b.tgClient.UnbanChatMember(userCfg)
		if err != nil {
			log.Printf("[WARN] failed to unban %s, %v", name, err)
			return Response{}
		}
		return Response{Text: fmt.Sprintf("амнистия для %s", name), Send: true}
	}

	return Response{}
}

func getUserID(name string, entities []Entity) int {
	if entities == nil {
		return 0
	}
	for _, e := range entities {
		if e.Type == "mention" && e.User != nil && e.User.DisplayName == name || e.User.Username == name {
			return e.User.ID
		}
	}
	return 0
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
