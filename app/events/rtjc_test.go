package events

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/events/mocks"
)

func TestRtjc_isPinned(t *testing.T) {
	tbl := []struct {
		inp string
		out string
		pin bool
	}{
		{"blah", "blah", false},
		{"⚠️ Официальный кАт! - https://stream.radio-t.com/", "⚠️ Вещание подкаста началось - https://stream.radio-t.com/", true},
		{" ⚠️ Официальный кАт! - https://stream.radio-t.com/ ", "⚠️ Вещание подкаста началось - https://stream.radio-t.com/", true},
		{" ⚠️ Официальный кАт! - https://stream.radio-t.com/\n", "⚠️ Вещание подкаста началось - https://stream.radio-t.com/", true},
	}

	rtjc := Rtjc{}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			pin, out := rtjc.isPinned(tt.inp)
			assert.Equal(t, tt.pin, pin)
			assert.Equal(t, tt.out, out)
		})
	}
}

func TestRtjc_summary(t *testing.T) {
	oai := &mocks.OpenAISummary{
		SummaryFunc: func(text string) (string, error) {
			return "ai summary", nil
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "token123", r.URL.Query().Get("token"))
		_, err := w.Write([]byte(`{"content": "some content", "title": "some title"}`))
		require.NoError(t, err)
	}))

	rtjc := Rtjc{OpenAISummary: oai, URClient: ts.Client(), UrAPI: ts.URL, UrToken: "token123"}

	{
		title, txt, err := rtjc.summary("some message blah https://example.com")
		require.NoError(t, err)
		assert.Equal(t, "ai summary", txt)
		assert.Equal(t, "some title - some content", oai.SummaryCalls()[0].Text)
		assert.Equal(t, "some title", title)
	}

	{
		title, txt, err := rtjc.summary("some message blah https://radio-t.com")
		require.NoError(t, err)
		assert.Equal(t, "", txt)
		assert.Equal(t, "", title)
	}
}
