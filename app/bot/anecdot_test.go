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
	mockHttp := &MockHttpClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(strings.NewReader("joke")),
	}, nil)

	response, answer := b.OnMessage(Message{Text: "joke!"})
	require.True(t, answer)
	require.Equal(t, "joke", response)
}

func TestAnecdot_ReactsOnJokeRequestAlt(t *testing.T) {
	mockHttp := &MockHttpClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(strings.NewReader("joke")),
	}, nil)

	response, answer := b.OnMessage(Message{Text: "/joke"})
	require.True(t, answer)
	require.Equal(t, "joke", response)
}

func TestAnecdot_RshunemaguRetursnNothingOnUnableToDoReq(t *testing.T) {
	mockHttp := &MockHttpClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(nil, errors.New("err"))

	response, answer := b.rzhunemogu()
	require.False(t, answer)
	require.Empty(t, response)
}

func TestAnecdotReactsOnUnexpectedMessage(t *testing.T) {
	mockHttp := &MockHttpClient{}
	b := NewAnecdote(mockHttp)

	result, answer := b.OnMessage(Message{Text: "unexpected msg"})
	require.False(t, answer)
	assert.Empty(t, result)
}

func TestAnecdotReactsOnBadChuckMessage(t *testing.T) {
	mockHttp := &MockHttpClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(`not a json`))),
	}, nil)

	result, answer := b.OnMessage(Message{Text: "chuck!"})
	require.False(t, answer)
	assert.Empty(t, result)
}

func TestAnecdotReactsOnChuckMessageUnableToDoReq(t *testing.T) {
	mockHttp := &MockHttpClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(nil, errors.New("err"))

	result, answer := b.OnMessage(Message{Text: "chuck!"})
	require.False(t, answer)
	assert.Empty(t, result)
}

func TestAnecdotReactsOnChuckMessage(t *testing.T) {
	mockHttp := &MockHttpClient{}
	b := NewAnecdote(mockHttp)

	mockHttp.On("Do", mock.Anything).Return(&http.Response{
		Body: ioutil.NopCloser(bytes.NewReader([]byte(`{"Value" : {"Joke" : "&quot;joke&quot;"}}`))),
	}, nil)

	result, answer := b.OnMessage(Message{Text: "chuck!"})
	require.True(t, answer)
	assert.Equal(t, "- \"joke\"", result)
}
