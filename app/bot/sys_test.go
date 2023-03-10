package bot

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSys_OnMessage(t *testing.T) {
	bot, err := NewSys("./../../data")
	require.NoError(t, err)
	rand.Seed(0) // nolint
	assert.Equal(t, Response{Text: "_никто не знает. пока не надоест_", Send: true}, bot.OnMessage(Message{Text: "доколе?"}))
	assert.Equal(t, Response{Text: "_понг_", Send: true}, bot.OnMessage(Message{Text: "пинг"}))
	assert.Equal(t, Response{Text: "_pong_", Send: true}, bot.OnMessage(Message{Text: "ping"}))
	assert.Equal(t, Response{Text: "_ Каждый французский солдат носит в своем ранце маршальский жезл._", Send: true}, bot.OnMessage(Message{Text: "Say!"}))
}

func TestSys_Help(t *testing.T) {
	bot, err := NewSys("./../../data")
	require.NoError(t, err)
	assert.Equal(t, "say! _– набраться мудрости_\n"+
		"ping _– ответит pong_\n"+
		"пинг _– ответит понг_\n"+
		"кто?, who? _– ведущие Радио-Т_\n"+
		"как?, how? _– online вещание Радио-Т_\n"+
		"доколе? _– день закрытия Радио-Т_\n"+
		"правила, rules?, правила? _– правила общения в чате_\n",
		bot.Help())
}

func TestSys_Failed(t *testing.T) {
	_, err := NewSys("/tmp/no-such-place")
	require.Error(t, err)
}
