package bot

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/radio-t/super-bot/app/bot/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewsBot_ReactionOnNewsRequest(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewNews(mockHTTP, "", 5)

	articles := []newsArticle{
		{
			Title: "title1",
			Link:  "link1",
			Ts:    time.Date(2020, 2, 9, 18, 45, 44, 0, time.UTC),
		},
		{
			Title: "title2",
			Link:  "link2",
			Ts:    time.Date(2020, 2, 10, 18, 45, 45, 0, time.UTC),
		},
	}
	articleJSON, err := json.Marshal(articles)
	require.NoError(t, err)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader(articleJSON)),
	}, nil)

	require.Equal(
		t,
		Response{Text: "- [title1](link1) 2020-02-09\n- [title2](link2) 2020-02-10" +
			"\n- [все новости и темы](https://news.radio-t.com)", Send: true},
		b.OnMessage(Message{Text: "news!"}),
	)
}

func TestNewsBot_ReactionOnNewsRequestAlt(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewNews(mockHTTP, "", 5)

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

func TestNewsBot_ReactionOnUnexpectedMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewNews(mockHTTP, "", 5)
	require.Equal(t, Response{}, b.OnMessage(Message{Text: "unexpected"}))
}
