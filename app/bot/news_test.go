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
	mockHTTP := &MockHTTPClient{}
	b := NewNews(mockHTTP, "")

	article := newsArticle{
		Title: "title",
		Link:  "link",
	}
	articleJSON, err := json.Marshal([]newsArticle{article})
	require.NoError(t, err)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(articleJSON)),
	}, nil)

	require.Equal(
		t,
		Response{Text: "- [title](link) 0001-01-01\n- [все новости и темы](https://news.radio-t.com)", Send: true},
		b.OnMessage(Message{Text: "news!"}),
	)
}

func TestNewsBot_ReactionOnNewsRequestAlt(t *testing.T) {
	mockHTTP := &MockHTTPClient{}
	b := NewNews(mockHTTP, "")

	article := newsArticle{
		Title: "title",
		Link:  "link",
	}
	articleJSON, err := json.Marshal([]newsArticle{article})
	require.NoError(t, err)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(articleJSON)),
	}, nil)

	require.Equal(
		t,
		Response{Text: "- [title](link) 0001-01-01\n- [все новости и темы](https://news.radio-t.com)", Send: true},
		b.OnMessage(Message{Text: "/news"}),
	)
}

func TestNewsBot_ReactionOnUnexpectedMessage(t *testing.T) {
	mockHTTP := &MockHTTPClient{}
	b := NewNews(mockHTTP, "")
	require.Equal(t, Response{}, b.OnMessage(Message{Text: "unexpected"}))
}
