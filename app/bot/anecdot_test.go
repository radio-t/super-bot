package bot

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestAnecdot_Help(t *testing.T) {
	a := NewAnecdote(http.DefaultClient)
	require.Equal(t, "анекдот!, анкедот!, joke!, chuck!, excuse!, pirozhki!, radiot!, zaibatsu!, excuse_en!, facts!, oneliner! _– расскажет анекдот или шутку_\n",
		a.Help())
}

func TestAnecdot_ReactsOnJokeRequest(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: io.NopCloser(strings.NewReader(`{"content": "Добраться до вершины не так сложно, как пробраться через толпу у её основания."}`)),
	}, nil)

	response := b.OnMessage(Message{Text: "joke!"})
	require.True(t, response.Send)
	require.Equal(t, "Добраться до вершины не так сложно, как пробраться через толпу у её основания", response.Text)
}

func TestAnecdot_ujokesrvRetursnNothingOnUnableToDoReq(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(nil, fmt.Errorf("err"))

	response := b.jokesrv("oneliners")
	require.False(t, response.Send)
	require.Empty(t, response.Text)
}

func TestAnecdotReactsOnUnexpectedMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(nil, fmt.Errorf("err"))

	result := b.OnMessage(Message{Text: "unexpected msg"})
	require.False(t, result.Send)
	assert.Empty(t, result.Text)
}

func TestAnecdotReactsOnBadChuckMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(`not a json`))),
	}, nil)

	require.Equal(t, Response{}, b.OnMessage(Message{Text: "chuck!"}))
}

func TestAnecdotReactsOnChuckMessageUnableToDoReq(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(nil, fmt.Errorf("err"))

	require.Equal(t, Response{}, b.OnMessage(Message{Text: "chuck!"}))
}

func TestAnecdotReactsOnChuckMessage(t *testing.T) {
	mockHTTP := &mocks.HTTPClient{}
	b := NewAnecdote(mockHTTP)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body: io.NopCloser(bytes.NewReader([]byte(`{"Value" : {"Joke" : "&quot;joke&quot;"}}`))),
	}, nil)

	require.Equal(t, Response{Text: "- \"joke\"", Send: true}, b.OnMessage(Message{Text: "chuck!"}))
}
