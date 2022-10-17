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
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}
	min := time.Hour * 24
	max := 7 * time.Hour * 24
	b := NewWTF(min, max, su)
	b.rand = func(n int64) int64 { return 10 }

	t.Run("not a wtf message", func(t *testing.T) {
		resp := b.OnMessage(Message{Text: "unrelevant text"})
		assert.Empty(t, resp)
	})

	t.Run("regular user, first wtf", func(t *testing.T) {
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user_with_underscores", ID: 1}})
		require.Equal(t, "@user\\_with\\_underscores получает бан на 1дн 10сек", resp.Text)
		require.True(t, resp.Send)
		require.Equal(t, min+10*time.Second, resp.BanInterval)
		assert.Equal(t, User{Username: "user_with_underscores", ID: 1}, resp.User)
	})

	t.Run("regular user, second wtf", func(t *testing.T) {
		resp := b.OnMessage(Message{Text: "WTF!", From: User{DisplayName: "user display name", ID: 1}})
		require.Equal(t, "user display name получает бан на 13дн 7ч 10сек", resp.Text)
		require.True(t, resp.Send)
		require.Equal(t, min+10*time.Second+59*5*time.Hour, resp.BanInterval)
	})

	t.Run("regular user, third wtf, some time passed since last wtf", func(t *testing.T) {
		time.Sleep(time.Second * 5)
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user"}})
		require.Equal(t, "@user получает бан на 12дн 6ч 10сек", resp.Text)
		require.True(t, resp.Send)
		require.Equal(t, min+10*time.Second+5*54*time.Hour, resp.BanInterval)
	})

	t.Run("channel, first wtf", func(t *testing.T) {
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "ChannelBot", ID: 136817688}, SenderChat: SenderChat{ID: 123, UserName: "channel_with_underscores"}})
		require.Equal(t, "@channel\\_with\\_underscores получает бан на навсегда", resp.Text)
		require.True(t, resp.Send)
		require.Equal(t, min+10*time.Second+59*5*time.Hour, resp.BanInterval)
		assert.Equal(t, User{Username: "ChannelBot", ID: 136817688}, resp.User)
		assert.Equal(t, int64(123), resp.ChannelID)
	})

	t.Run("admin, allow wtf", func(t *testing.T) {
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "super"}})
		require.Equal(t, "", resp.Text)
		require.False(t, resp.Send)
		require.Equal(t, time.Duration(0), resp.BanInterval)
	})

	t.Run("admin, reply wtf", func(t *testing.T) {
		msg := Message{Text: "WTF!", From: User{Username: "super"}}
		msg.ReplyTo.From = User{Username: "user", ID: 1, DisplayName: "User"}
		resp := b.OnMessage(msg)
		assert.Equal(t, "@user получает бан на 13дн 7ч 10сек", resp.Text)
		assert.True(t, resp.Send)
		assert.Equal(t, min+10*time.Second+5*59*time.Hour, resp.BanInterval)
	})

	t.Run("admin, reply wtf to channel", func(t *testing.T) {
		msg := Message{Text: "WTF!", From: User{Username: "super"}}
		msg.ReplyTo.From = User{Username: "ChannelBot", ID: 136817688}
		msg.ReplyTo.SenderChat = SenderChat{ID: 123, UserName: "channel"}
		resp := b.OnMessage(msg)
		assert.Equal(t, "@channel получает бан на навсегда", resp.Text)
		assert.True(t, resp.Send)
		assert.Equal(t, min+10*time.Second+5*59*time.Hour, resp.BanInterval)
	})

	t.Run("admin, reply wtf to another admin", func(t *testing.T) {
		msg := Message{Text: "WTF!", From: User{Username: "super"}}
		msg.ReplyTo.From = User{Username: "admin", ID: 321}
		resp := b.OnMessage(msg)
		assert.Empty(t, resp)
	})

	t.Run("regular user, magic wtf duration #1", func(t *testing.T) {
		b.rand = func(n int64) int64 { return 1 }
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user"}})
		require.Equal(t, "@user получает бан на 27дн 18ч (666 часов)", resp.Text)
		require.True(t, resp.Send)
		require.Equal(t, time.Hour*666, resp.BanInterval)
	})

	t.Run("regular user, magic wtf duration #2", func(t *testing.T) {
		b.rand = func(n int64) int64 { return 2 }
		resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user"}})
		require.Equal(t, "@user получает бан на 1ч 17мин 7сек", resp.Text)
		require.True(t, resp.Send)
		require.Equal(t, time.Minute*77+time.Second*7, resp.BanInterval)
	})
	assert.Equal(t, 19, len(su.IsSuperCalls()))
}

func TestWTF_Help(t *testing.T) {
	require.Equal(t, "wtf!, wtf? _– если не повезет, блокирует пользователя на какое-то время_\n", (&WTF{}).Help())
}
