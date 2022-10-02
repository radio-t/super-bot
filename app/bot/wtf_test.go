package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestWTF_OnMessage(t *testing.T) {
	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" {
			return true
		}
		return false
	}}
	min := time.Hour * 24
	max := 7 * time.Hour * 24
	b := NewWTF(min, max, su)
	b.rand = func(n int64) int64 {
		return 10
	}

	{ // regular user, first wtf
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user", ID: 1}})
		require.Equal(t, "[@user](tg://user?id=1) получает бан на 1дн 10сек", resp.Text)
		require.True(t, resp.Send)
		require.Equal(t, min+10*time.Second, resp.BanInterval)
		assert.Equal(t, User{Username: "user", ID: 1}, resp.User)
	}

	{ // regular user, second wtf
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user", ID: 1}})
		require.Equal(t, "[@user](tg://user?id=1) получает бан на 13дн 7ч 10сек", resp.Text)
		require.True(t, resp.Send)
		require.Equal(t, min+10*time.Second+59*5*time.Hour, resp.BanInterval)
	}

	{ // regular user, third wtf, some time passed since last wtf
		time.Sleep(time.Second * 5)
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user"}})
		require.Equal(t, "[@user](tg://user?id=0) получает бан на 12дн 6ч 10сек", resp.Text)
		require.True(t, resp.Send)
		require.Equal(t, min+10*time.Second+5*54*time.Hour, resp.BanInterval)
	}

	{ // admin, allow wtf
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "super"}})
		require.Equal(t, "", resp.Text)
		require.False(t, resp.Send)
		require.Equal(t, time.Duration(0), resp.BanInterval)
	}

	{ // admin, reply wtf
		msg := Message{Text: "WTF!", From: User{Username: "super"}}
		msg.ReplyTo.From = User{Username: "user", ID: 1, DisplayName: "User"}
		resp := b.OnMessage(msg)
		assert.Equal(t, "[@user](tg://user?id=1) получает бан на 13дн 7ч 10сек", resp.Text)
		assert.True(t, resp.Send)
		assert.Equal(t, min+10*time.Second+5*59*time.Hour, resp.BanInterval)
	}
	assert.Equal(t, 9, len(su.IsSuperCalls()))
}

func TestWTF_Help(t *testing.T) {
	require.Equal(t, "wtf!, wtf? _– если не повезет, блокирует пользователя на какое-то время_\n", (&WTF{}).Help())
}
