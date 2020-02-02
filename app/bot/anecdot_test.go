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
)

func TestAnecdot_ReactsOnJokeRequest(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(strings.NewReader("joke")),
	}, nil)

	response := b.OnMessage(Message{Text: "joke!"})
	require.True(t, response.Send)
	require.Equal(t, "joke", response.Text)
}

func TestAnecdot_ReactsOnJokeRequestAlt(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(strings.NewReader("joke")),
	}, nil)

	response := b.OnMessage(Message{Text: "/joke"})
	require.True(t, response.Send)
	require.Equal(t, "joke", response.Text)
}

func TestAnecdot_RshunemaguRetursnNothingOnUnableToDoReq(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(nil, errors.New("err"))

	response := b.rzhunemogu()
	require.False(t, response.Send)
	require.Empty(t, response.Text)
}

func TestAnecdotReactsOnUnexpectedMessage(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewAnecdote(mockHttp)

	result := b.OnMessage(Message{Text: "unexpected msg"})
	require.False(t, result.Send)
	assert.Empty(t, result.Text)
}

func TestAnecdotReactsOnBadChuckMessage(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(`not a json`))),
	}, nil)

	result := b.OnMessage(Message{Text: "chuck!"})
	require.False(t, result.Send)
	assert.Empty(t, result.Text)
}

func TestAnecdotReactsOnChuckMessageUnableToDoReq(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(nil, errors.New("err"))

	result := b.OnMessage(Message{Text: "chuck!"})
	require.False(t, result.Send)
	assert.Empty(t, result.Text)
}

func TestAnecdotReactsOnChuckMessage(t *testing.T) {
	mockHttp := &MockHTTPClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"Value" : {"Joke" : "&quot;joke&quot;"}}`))),
	}, nil)

	result := b.OnMessage(Message{Text: "chuck!"})
	require.True(t, result.Send)
	assert.Equal(t, "- \"joke\"", result.Text)
}
