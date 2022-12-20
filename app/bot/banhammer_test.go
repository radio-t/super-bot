package bot

import (
	"strconv"
	"testing"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestBanhammer_Help(t *testing.T) {
	b := NewBanhammer(nil, nil, 10)
	assert.Equal(t, "ban!, unban! _– забанить/разбанить (только для админов)_\n", b.Help())
}

func TestBanhammer_parse(t *testing.T) {

	tbl := []struct {
		text string
		ok   bool
		cmd  string
		req  string
	}{
		{"blah", false, "", ""},
		{"ban!someone", true, "ban", "someone"},
		{"ban! user2", true, "ban", "user2"},
		{"unban! user2", true, "unban", "user2"},
	}

	b := &Banhammer{}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ok, cmd, req := b.parse(tt.text)
			if !tt.ok {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			assert.Equal(t, tt.cmd, cmd)
			assert.Equal(t, tt.req, req)
		})
	}
}

func TestBanhammer_OnMessage(t *testing.T) {
	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "admin" {
			return true
		}
		return false
	}}
	tg := &mocks.TgBanClient{RequestFunc: func(c tbapi.Chattable) (*tbapi.APIResponse, error) {
		return &tbapi.APIResponse{Ok: true}, nil
	}}
	b := NewBanhammer(tg, su, 10)

	resp := b.OnMessage(Message{Text: "ban! user1", From: User{Username: "user1", ID: 1}})
	assert.Equal(t, Response{}, resp, "not admin")

	resp = b.OnMessage(Message{Text: "bawwwn! user1", From: User{Username: "admin", ID: 0}})
	assert.Equal(t, Response{}, resp, "not a command")

	resp = b.OnMessage(Message{Text: "ban! user1", From: User{Username: "admin"}, ChatID: 123})
	assert.Equal(t, Response{Text: "прощай user1", Send: true}, resp)

	resp = b.OnMessage(Message{Text: "unban! user1", From: User{Username: "admin"}, ChatID: 123})
	assert.Equal(t, Response{Text: "амнистия для user1", Send: true}, resp)

	assert.Equal(t, 5, len(su.IsSuperCalls()))
	assert.Equal(t, 2, len(tg.RequestCalls()))
	assert.Equal(t, int64(1), tg.RequestCalls()[0].C.(tbapi.BanChatMemberConfig).UserID)
	assert.Equal(t, int64(123), tg.RequestCalls()[0].C.(tbapi.BanChatMemberConfig).ChatID)
	assert.Equal(t, int64(1), tg.RequestCalls()[1].C.(tbapi.UnbanChatMemberConfig).UserID)
	assert.Equal(t, int64(123), tg.RequestCalls()[1].C.(tbapi.UnbanChatMemberConfig).ChatID)
}
