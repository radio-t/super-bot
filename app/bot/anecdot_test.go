package bot

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestAnecdot_Help(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(`[
	"excuse",
	"pirozhki",
	"radiot",
	"zaibatsu",
	"excuse_en",
	"facts",
	"oneliner"
]`))}, nil
	}}
	a := NewAnecdote(mockHTTP)
	require.Equal(t, "анекдот!, анкедот!, joke!, chuck!, excuse!, pirozhki!, radiot!, zaibatsu!, excuse\\_en!, facts!, oneliner! _– расскажет анекдот или шутку_\n",
		a.Help())
}

func TestAnecdot_ReactsOnJokeRequest(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body: io.NopCloser(strings.NewReader(`{"content": "Добраться до вершины не так сложно, как пробраться через толпу у её основания."}`)),
		}, nil
	}}
	b := NewAnecdote(mockHTTP)

	response := b.OnMessage(Message{Text: "joke!"})
	require.True(t, response.Send)
	require.Equal(t, "Добраться до вершины не так сложно, как пробраться через толпу у её основания", response.Text)
}

func TestAnecdot_ujokesrvRetursnNothingOnUnableToDoReq(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("err")
	}}
	b := NewAnecdote(mockHTTP)

	response := b.jokesrv("oneliners")
	require.False(t, response.Send)
	require.Empty(t, response.Text)
}

func TestAnecdotReactsOnUnexpectedMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("err")
	}}
	b := NewAnecdote(mockHTTP)

	result := b.OnMessage(Message{Text: "unexpected msg"})
	require.False(t, result.Send)
	assert.Empty(t, result.Text)
}

func TestAnecdotReactsOnBadChuckMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(`not a json`))),
		}, nil
	}}
	b := NewAnecdote(mockHTTP)

	require.Equal(t, Response{}, b.OnMessage(Message{Text: "chuck!"}))
}

func TestAnecdotReactsOnChuckMessageUnableToDoReq(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("err")
	}}
	b := NewAnecdote(mockHTTP)

	require.Equal(t, Response{}, b.OnMessage(Message{Text: "chuck!"}))
}

func TestAnecdotReactsOnChuckMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(`{
"categories":[],
"created_at":"2020-01-05 13:42:23.880601",
"icon_url":"https://assets.chucknorris.host/img/avatar/chuck-norris.png",
"id":"Lf-jNoPpSZqXoTySlBCtfg",
"updated_at":"2020-01-05 13:42:23.880601",
"url":"https://api.chucknorris.io/jokes/Lf-jNoPpSZqXoTySlBCtfg",
"value":"Chuck Norris got pulled over by a cop once. The cop was lucky to leave with a _warning_."}
`))),
		}, nil
	}}
	b := NewAnecdote(mockHTTP)

	require.Equal(t, Response{Text: "Chuck Norris got pulled over by a cop once. The cop was lucky to leave with a \\_warning\\_.", Send: true}, b.OnMessage(Message{Text: "chuck!"}))
}
