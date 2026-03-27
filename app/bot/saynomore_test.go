package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestSayNoMore_OnMessage(t *testing.T) {
	mockSuperUser := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		return userName == "super"
	}}

	categories := []SayNoMoreCategory{
		{Words: []string{"badword", "verybad"}, Responses: []string{"ай-ай-ай", "как не стыдно"}},
		{Words: []string{"rude"}, Responses: []string{"грубиян"}},
	}

	bot := NewSayNoMore(time.Minute, time.Hour, mockSuperUser, categories)
	bot.rand = func(n int64) int64 { return 0 } // fixed for tests

	t.Run("exact match triggers ban", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "badword",
			From: User{Username: "user1", DisplayName: "User One"},
		})
		assert.True(t, resp.Send)
		assert.Contains(t, resp.Text, "@user1")
		assert.Contains(t, resp.Text, "ай-ай-ай") // first response from first category
		assert.Equal(t, time.Minute, resp.BanInterval)
	})

	t.Run("second category match", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "rude",
			From: User{Username: "user1"},
		})
		assert.True(t, resp.Send)
		assert.Contains(t, resp.Text, "грубиян")
	})

	t.Run("case insensitive match", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "BADWORD",
			From: User{Username: "user1"},
		})
		assert.True(t, resp.Send)
	})

	t.Run("no match returns empty", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "hello world",
			From: User{Username: "user1"},
		})
		assert.False(t, resp.Send)
	})

	t.Run("whole word in sentence triggers", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "this contains badword inside",
			From: User{Username: "user1"},
		})
		assert.True(t, resp.Send)
	})

	t.Run("embedded word does not trigger", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "thisbadwordhere",
			From: User{Username: "user1"},
		})
		assert.False(t, resp.Send)
	})

	t.Run("case insensitive whole word in sentence triggers", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "this contains BADWORD inside",
			From: User{Username: "user1"},
		})
		assert.True(t, resp.Send)
	})

	t.Run("super user is ignored", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "badword",
			From: User{Username: "super"},
		})
		assert.False(t, resp.Send)
	})

	t.Run("user without username uses display name", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "badword",
			From: User{DisplayName: "John Doe"},
		})
		assert.True(t, resp.Send)
		assert.Contains(t, resp.Text, "John Doe")
	})

	t.Run("random response is picked from category", func(t *testing.T) {
		bot2 := NewSayNoMore(time.Minute, time.Hour, mockSuperUser, categories)
		bot2.rand = func(n int64) int64 { return 1 } // pick second response

		resp := bot2.OnMessage(Message{
			Text: "badword",
			From: User{Username: "user1"},
		})
		assert.Contains(t, resp.Text, "как не стыдно")
	})
}

func TestSayNoMore_CyrillicMatching(t *testing.T) {
	mockSuperUser := &mocks.SuperUser{IsSuperFunc: func(userName string) bool { return false }}

	bot := NewDefaultSayNoMore(mockSuperUser)
	bot.rand = func(n int64) int64 { return 0 }

	t.Run("cyrillic phrase in sentence triggers", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "а когда крайний выпуск?",
			From: User{Username: "user1"},
		})
		assert.True(t, resp.Send)
	})

	t.Run("cyrillic exact match triggers", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "ихний",
			From: User{Username: "user1"},
		})
		assert.True(t, resp.Send)
	})

	t.Run("cyrillic case insensitive triggers", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "ИХНИЙ",
			From: User{Username: "user1"},
		})
		assert.True(t, resp.Send)
	})

	t.Run("cyrillic embedded word does not trigger", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "поихнийму",
			From: User{Username: "user1"},
		})
		assert.False(t, resp.Send)
	})

	t.Run("cyrillic word with punctuation triggers", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "это ихний!",
			From: User{Username: "user1"},
		})
		assert.True(t, resp.Send)
	})
}

func TestSayNoMore_EmptyResponses(t *testing.T) {
	mockSuperUser := &mocks.SuperUser{IsSuperFunc: func(userName string) bool { return false }}
	categories := []SayNoMoreCategory{
		{Words: []string{"badword"}, Responses: []string{}},
		{Words: []string{"rude"}, Responses: []string{"грубиян"}},
	}

	bot := NewSayNoMore(time.Minute, time.Hour, mockSuperUser, categories)
	bot.rand = func(n int64) int64 { return 0 }

	t.Run("category with empty responses is skipped", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "badword",
			From: User{Username: "user1"},
		})
		assert.False(t, resp.Send)
	})

	t.Run("category with responses still works", func(t *testing.T) {
		resp := bot.OnMessage(Message{
			Text: "rude",
			From: User{Username: "user1"},
		})
		assert.True(t, resp.Send)
	})
}

func TestSayNoMore_ReactOn(t *testing.T) {
	bot := NewSayNoMore(time.Minute, time.Hour, nil, nil)
	assert.Nil(t, bot.ReactOn())
}

func TestSayNoMore_Help(t *testing.T) {
	bot := NewSayNoMore(time.Minute, time.Hour, nil, nil)
	assert.Equal(t, "Боремся за чистоту чата", bot.Help())
}
