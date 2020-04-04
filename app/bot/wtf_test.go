package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWTF_OnMessage(t *testing.T) {
	b := NewWTF(time.Hour*24, 7*time.Hour*24)
	r := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user123"}})
	t.Logf("%v", r)
	assert.Contains(t, r.Text, "@user123")
	assert.True(t, r.Send)

	var totBanTime time.Duration
	for i := 0; i < 100; i++ {
		r = b.OnMessage(Message{Text: "wtf!", From: User{Username: "xyz"}})
		require.True(t, r.BanInterval > 0)
		assert.True(t, r.Send)
		totBanTime += r.BanInterval
	}
	t.Log(totBanTime / 100)
	assert.True(t, totBanTime/100 >= time.Hour*24)
}

func TestWTF_Help(t *testing.T) {
	require.Equal(t, "wtf!, wtf? _– если не повезет, блокирует пользователя на какое-то время_\n", (&WTF{}).Help())
}
