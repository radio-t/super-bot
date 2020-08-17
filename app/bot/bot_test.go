package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMultiBot_OnMessage(t *testing.T) {
	bb1 := &MockInterface{}
	bb2 := &MockInterface{}
	bb3 := &MockInterface{}

	b := MultiBot{bb1, bb2, bb3}

	bb1.On("OnMessage", mock.MatchedBy(func(msg Message) bool {
		require.Equal(t, "message", msg.Text)
		return true
	})).Return(Response{Text: "response1", Send: true})

	bb2.On("OnMessage", mock.MatchedBy(func(msg Message) bool {
		require.Equal(t, "message", msg.Text)
		return true
	})).Return(Response{Text: "response2", Send: true})

	bb3.On("OnMessage", mock.MatchedBy(func(msg Message) bool {
		require.Equal(t, "message", msg.Text)
		return true
	})).Return(Response{Text: "response3", Send: false})

	resp := b.OnMessage(Message{
		ID:     1,
		From:   User{Username: "name"},
		ChatID: 123,
		Sent:   time.Now(),
		Text:   "message",
	})

	assert.Equal(t, Response{Text: "response1\nresponse2", Send: true, Pin: false, Unpin: false, Preview: false, BanInterval: 0}, resp)

	bb1.AssertExpectations(t)
	bb2.AssertExpectations(t)
	bb3.AssertExpectations(t)
}

func TestMultiBot_ReactOn(t *testing.T) {
	bb1 := &MockInterface{}
	bb2 := &MockInterface{}
	bb3 := &MockInterface{}

	bb1.On("ReactOn").Return([]string{"r11", "r12"})
	bb2.On("ReactOn").Return([]string{"r21", "r22", "r23"})
	bb3.On("ReactOn").Return([]string{"r31"})

	b := MultiBot{bb1, bb2, bb3}

	assert.Equal(t, []string{"r11", "r12", "r21", "r22", "r23", "r31"}, b.ReactOn())
}

func TestMultiBot_Help(t *testing.T) {
	bb1 := &MockInterface{}
	bb2 := &MockInterface{}
	bb3 := &MockInterface{}

	bb1.On("Help").Return("h1")
	bb2.On("Help").Return("h2\n")
	bb3.On("Help").Return("h3")

	b := MultiBot{bb1, bb2, bb3}

	assert.Equal(t, "h1\nh2\nh3\n", b.Help())
}
