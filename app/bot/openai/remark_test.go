package openai

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemarkClient_GetTopComments(t *testing.T) {
	jsonResponse, err := os.ReadFile("testdata/remark_response.json")
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, err = w.Write(jsonResponse)
		require.NoError(t, err)
	}))

	rc := RemarkClient{
		Client: ts.Client(),
		API:    ts.URL,
	}

	comments, _, err := rc.GetTopComments("http://radio-t.com")
	assert.NoError(t, err)
	require.Equal(t, 4, len(comments))
	assert.Equal(t, "<b>+3</b> от <b>user2</b>\n<i>some comment 3</i>", comments[0])
	assert.Equal(t, "<b>+2</b> от <b>user4</b>\n<i>some comment 5</i>", comments[1])
	assert.Equal(t, "<b>+2</b> от <b>user3</b>\n<i>some comment 4</i>", comments[2])
	assert.Equal(t, "<b>+1</b> от <b>user1</b>\n<i>some comment 1</i>", comments[3])
}

func TestRemarkComment_GetLink(t *testing.T) {
	tbl := []struct {
		name     string
		text     string
		expected string
	}{
		{"Good link", `We have a <a href="https://podcast.umputun.com">link</a>`, "https://podcast.umputun.com"},
		{"Good link after image", `We have <img src="https://image.com/img.jpeg"> a link <a href="https://podcast.umputun.com">link</a>`, "https://podcast.umputun.com"},
		{"Bad link", `We have a <a href="https://podcast.umputun.com</a>`, ""},
		{"No link", `Just text`, ""},
		{"Radio-t.com link", `We have a link <a href="https://radio-t.com/p/hello">link</a>`, ""},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			comment := remarkComment{
				Text: tt.text,
			}

			assert.Equal(t, tt.expected, comment.getLink())
		})
	}
}

func TestRemarkComment_Render(t *testing.T) {
	tbl := []struct {
		name     string
		user     string
		text     string
		score    int
		expected string
	}{
		{"Simple", "user1", "some text 1", 1, "<b>+1</b> от <b>user1</b>\n<i>some text 1</i>"},
		{"Zero score", "user2", "some text 2", 0, "<b>+0</b> от <b>user2</b>\n<i>some text 2</i>"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			comment := remarkComment{
				Text: tt.text,
				User: struct {
					Name     string `json:"name"`
					Admin    bool   `json:"admin"`
					Verified bool   `json:"verified,omitempty"`
				}{tt.user, false, false},
				Score: tt.score,
			}

			assert.Equal(t, tt.expected, comment.render())
		})
	}
}
