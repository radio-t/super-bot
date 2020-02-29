package bot

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWTF_OnMessage(t *testing.T) {
	b := NewWTF(time.Minute, 10*time.Minute, 0.2, 0.1)
	r := b.OnMessage(Message{Text: "WTF!", From: User{Username: "user123"}})
	t.Logf("%v", r)
	assert.Contains(t, r.Text, "@user123")
	assert.True(t, r.Send)

	unlucky, lucky := 0, 0
	for i := 0; i < 100; i++ {
		r = b.OnMessage(Message{Text: "wtf!", From: User{Username: "xyz"}})
		assert.True(t, r.Send)
		switch {
		case strings.Contains(r.Text, "повезло"):
			lucky++
			require.True(t, r.BanInterval == 0)
			t.Logf("%v", r)
		case strings.Contains(r.Text, "не твой день"):
			unlucky++
			require.True(t, r.BanInterval == 24*time.Hour)
			t.Logf("%v", r)
		default:
			require.True(t, r.BanInterval > 0)
		}
	}
	t.Logf("lucky hits %d", lucky)
	assert.True(t, lucky > 0)
	assert.True(t, unlucky > 0)
	assert.True(t, lucky < 100)
}

func TestWTF_Help(t *testing.T) {
	require.Equal(t, "wtf!, wtf? _– если не повезет, блокирует пользователя на какое-то время_\n", (&WTF{}).Help())
}
