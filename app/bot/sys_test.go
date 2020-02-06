package bot

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSys_OnMessage(t *testing.T) {
	bot := NewSys("./../../data")

	assert.Equal(t, Response{Text: "_никто не знает. пока не надоест_", Send: true}, bot.OnMessage(Message{Text: "доколе?"}))
	assert.Equal(t, Response{Text: "_понг_", Send: true}, bot.OnMessage(Message{Text: "пинг"}))
	assert.Equal(t, Response{Text: "_pong_", Send: true}, bot.OnMessage(Message{Text: "ping"}))
}
