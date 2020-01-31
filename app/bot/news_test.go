package bot

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewsBot_ReactionOnNewsRequest(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewNews(mockHttp, "")

	article := newsArticle{
		Title: "title",
		Link:  "link",
	}
	articleJson, err := json.Marshal([]newsArticle{article})
	require.NoError(t, err)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(articleJson)),
	}, nil)

	response, answer := b.OnMessage(Message{Text: "news!"})
	require.True(t, answer)
	require.Equal(t, "- [title](link) 0001-01-01\n- [все новости и темы](https://news.radio-t.com)", response)
}

func TestNewsBot_ReactionOnNewsRequestAlt(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewNews(mockHttp, "")

	article := newsArticle{
		Title: "title",
		Link:  "link",
	}
	articleJson, err := json.Marshal([]newsArticle{article})
	require.NoError(t, err)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(articleJson)),
	}, nil)

	response, answer := b.OnMessage(Message{Text: "/news"})
	require.True(t, answer)
	require.Equal(t, "- [title](link) 0001-01-01\n- [все новости и темы](https://news.radio-t.com)", response)
}

func TestNewsBot_ReactionOnUnexpectedMessage(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewNews(mockHttp, "")

	response, answer := b.OnMessage(Message{Text: "unexpected"})
	require.False(t, answer)
	require.Empty(t, response)
}
