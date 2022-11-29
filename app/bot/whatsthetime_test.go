package bot

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWhatsTheTime_OnMessage(t *testing.T) {
	bot, err := NewWhatsTheTime("./../../data")
	require.NoError(t, err)
	tbl := bot.ReactOn()
	expectedResult := `У Ksenks сейчас \d{2}:\d{2}\nУ Umputun сейчас \d{2}:\d{2}\nУ Bobuk сейчас \d{2}:\d{2}\nУ Gray сейчас \d{2}:\d{2}\nУ Alek.sys сейчас \d{2}:\d{2}`
	for i, command := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			msg := Message{Text: command}
			res := bot.OnMessage(msg).Text
			matched, err := regexp.MatchString(expectedResult,res)
			require.NoError(t, err)
			require.True(t, matched)
		})
	}
}

func TestWhatsTheTime_Help(t *testing.T) {
	b, err := NewWhatsTheTime("./../../data")
	require.NoError(t, err)
	require.Equal(t, "время!, time!, который час? _– подcкажет время у ведущих_\n", b.Help())
}