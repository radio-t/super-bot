package bot

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnecdot_BanInterval(t *testing.T) {
	b := MultiBot{
		NewWTF(time.Minute, 10*time.Minute, 0.2),
	}

	lucky := 0
	for i := 0; i < 100; i++ {
		r := b.OnMessage(Message{Text: "wtf!", From: User{Username: "user123"}})
		assert.True(t, r.Send)
		if strings.Contains(r.Text, "повезло") {
			lucky++
			require.True(t, r.BanInterval == 0)
		} else {
			require.True(t, r.BanInterval > 0)
		}
	}
	t.Logf("lucky hits %d", lucky)
	assert.True(t, lucky > 0)
	assert.True(t, lucky < 100)
}
