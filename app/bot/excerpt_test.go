package bot

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExcerpt_Link(t *testing.T) {

	tbl := []struct {
		inp  string
		link string
		fail bool
	}{
		{"blah", "", true},
		{"blah http://radio-t.com blah2", "http://radio-t.com", false},
		{"https://radio-t.com blah2", "https://radio-t.com", false},
		{"blah http://radio-t.com/aa.gif blah2", "", true},
		{"blah https://radio-t.com/aa.png blah2", "", true},
		{"blah https://radio-t.com/png blah2", "https://radio-t.com/png", false},
		{"blah https://twitter.com/radio_t/status/811670832510537730", "", true},
	}

	ex := NewExcerpt("http://parser.ukeeper.com/api/content/v1/parser", "")
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			link, err := ex.link(tt.inp)
			if tt.fail {
				require.NotNil(t, err)
			}
			assert.Equal(t, tt.link, link)
		})
	}
}

func TestExcerpt(t *testing.T) {

	tbl := []struct {
		link    string
		excerpt string
		fail    bool
	}{
		{"https://radio-t.com/p/2016/11/06/bot/", "В выпуске 520 была озвучена идея “сделай своего бота для любимого подкаста”." +
			" Я создал репо для этого дела где попытался описать как и что. Надеюсь, " +
			"получилось понятно. В двух словах - каждый ваш бот это микро-рест запакованный в контейнер и " +
			"получающий все сообщения из нашего чата. Если боту есть ...\n\n" +
			"_Больше ботов, хороших и разных — Радио-Т Подкаст_", false},
		{"https://xxxx.radio-t.com blah2", "", true},
	}

	ex := NewExcerpt("http://parser.ukeeper.com/api/content/v1/parser", "123456")

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r := ex.OnMessage(Message{Text: tt.link})
			require.Equal(t, tt.fail, !r.Send)
			if !tt.fail {
				assert.Equal(t, tt.excerpt, r.Text)
			}
		})
	}

}
