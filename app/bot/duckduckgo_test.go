package bot

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestDuck_Help(t *testing.T) {
	require.Equal(t, "ddg!, ?? _– поискать на DuckDuckGo, например: ddg! lambda_\n", (&Duck{}).Help())
}

func TestDuck_OnMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(`{"AbstractText":"the answer", "AbstractSource":"test", "AbstractURL":"http://example.com"}`))),
		}, nil
	}}
	d := NewDuck("key", mockHTTP)

	assert.Equal(t, Response{Text: "the answer\n[test](http://example.com)", Send: true}, d.OnMessage(Message{Text: "?? search"}))
}

func TestDuck_request(t *testing.T) {

	tbl := []struct {
		text string
		ok   bool
		req  string
	}{
		{"blah", false, ""},
		{"?? something", true, "something"},
		{"ddg! something", true, "something"},
	}

	d := &Duck{}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ok, req := d.request(tt.text)
			if !tt.ok {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			assert.Equal(t, tt.req, req)
		})
	}
}
