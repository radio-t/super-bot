package bot

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDuck_Help(t *testing.T) {
	require.Equal(t, "/ddg, ddg!, ?? _- поискать на DuckDuckGo, например: /ddg lambda_\n", (&Duck{}).Help())
}

func TestDuck_OnMessage(t *testing.T) {
	mockHTTP := &MockHTTPClient{}
	d := NewDuck("key", mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"AbstractText":"the answer", "AbstractSource":"test", "AbstractURL":"http://example.com"}`))),
	}, nil)

	assert.Equal(t, Response{Text: "- the answer\n[test](http://example.com)", Send: true}, d.OnMessage(Message{Text: "?? search"}))
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
		{"/ddg something", true, "something"},
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
