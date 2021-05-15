package bot

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestAnecdot_Help(t *testing.T) {
	require.Equal(t, "анекдот!, анкедот!, joke!, chuck!, facts!, zaibatsu! _– расскажет анекдот или шутку_\n",
		Anecdote{}.Help())
}

func TestAnecdot_ReactsOnJokeRequest(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(strings.NewReader(`{"content": "Добраться до вершины не так сложно, как пробраться через толпу у её основания.."}`)),
	}, nil)

	response := b.OnMessage(Message{Text: "joke!"})
	require.True(t, response.Send)
	require.Equal(t, "Добраться до вершины не так сложно, как пробраться через толпу у её основания..", response.Text)
}

func TestAnecdot_ujokesrvRetursnNothingOnUnableToDoReq(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(nil, errors.New("err"))

	response := b.jokesrv("oneliners")
	require.False(t, response.Send)
	require.Empty(t, response.Text)
}

func TestAnecdotReactsOnUnexpectedMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	result := b.OnMessage(Message{Text: "unexpected msg"})
	require.False(t, result.Send)
	assert.Empty(t, result.Text)
}

func TestAnecdotReactsOnBadChuckMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(`not a json`))),
	}, nil)

	require.Equal(t, Response{}, b.OnMessage(Message{Text: "chuck!"}))
}

func TestAnecdotReactsOnChuckMessageUnableToDoReq(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(nil, errors.New("err"))

	require.Equal(t, Response{}, b.OnMessage(Message{Text: "chuck!"}))
}

func TestAnecdotReactsOnChuckMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"Value" : {"Joke" : "&quot;joke&quot;"}}`))),
	}, nil)

	require.Equal(t, Response{Text: "- \"joke\"", Send: true}, b.OnMessage(Message{Text: "chuck!"}))
}
