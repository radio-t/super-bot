package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestName(t *testing.T) {
	min := time.Hour * 24
	max := 7 * time.Hour * 24
	b := NewWTF(min, max)
	b.rand = func(n int64) int64 {
		return 10
	}

	resp := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user"}})
	require.Equal(t, "[@user](tg://user?id=0) получает бан на 1дн 10сек", resp.Text)
	require.True(t, resp.Send)
	require.Equal(t, min+10*time.Second, resp.BanInterval)
}

func TestWTF_Help(t *testing.T) {
	require.Equal(t, "wtf!, wtf? _– если не повезет, блокирует пользователя на какое-то время_\n", (&WTF{}).Help())
}
