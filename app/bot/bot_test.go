package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAnecdot_BanInterval(t *testing.T) {
	b := MultiBot{
		NewWTF(time.Minute, 10*time.Minute),
	}

	for i := 0; i < 100; i++ {
		r := b.OnMessage(Message{Text: "wtf!", From: User{Username: "user123"}})
		assert.True(t, r.Send)
	}
}
