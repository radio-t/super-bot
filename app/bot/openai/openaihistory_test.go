package openai

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	ai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot"
)

func Test_LimitedMessageHistory(t *testing.T) {
	tests := []struct {
		name  string
		limit int
	}{
		{name: "Limit 5", limit: 5},
		{name: "Limit 10", limit: 10},
		{name: "Limit 20", limit: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			history := NewLimitedMessageHistory(tt.limit)

			// add `limit` messages to the storage
			for i := 0; i < tt.limit; i++ {
				history.Add(bot.Message{
					ID:   i,
					Text: fmt.Sprintf("test %d", i),
				})

				assert.Equal(t, i+1, len(history.messages))
			}

			assert.Equal(t, tt.limit, len(history.messages))
			for i := 0; i < tt.limit; i++ {
				assert.Equal(t, i, history.messages[i].ID)
				assert.Equal(t, fmt.Sprintf("test %d", i), history.messages[i].Text)
			}

			// add messages to the storage. This should remove the oldest messages
			for j := range 3 {
				newID := tt.limit + j
				history.Add(bot.Message{
					ID:   newID,
					Text: fmt.Sprintf("test %d", newID),
				})

				assert.Equal(t, tt.limit, len(history.messages))
				for i := 0; i < tt.limit; i++ {
					expectedID := i + j + 1
					assert.Equal(t, expectedID, history.messages[i].ID)
					assert.Equal(t, fmt.Sprintf("test %d", expectedID), history.messages[i].Text)
				}
			}

		})
	}

}

func TestLimitedMessageHistory_snapshot(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	cur := bot.Message{ID: 9, ChatID: -100, Sent: now, Text: "current request"}
	history := NewLimitedMessageHistory(20)
	for _, msg := range []bot.Message{
		{ID: 1, ChatID: -100, Sent: now.Add(-30*time.Minute - time.Nanosecond), Text: "expired"},
		{ID: 2, ChatID: -100, Sent: now.Add(-30 * time.Minute), Text: "boundary"},
		{ID: 3, ChatID: -200, Sent: now.Add(-time.Minute), Text: "other chat"},
		{ID: 4, ChatID: -100, Sent: now.Add(-time.Minute), Text: "recent"},
		{ID: 5, ChatID: -100, Text: "unknown age"},
		cur,
		{ID: 10, ChatID: -100, Sent: now.Add(time.Second), Text: "future"},
	} {
		history.Add(msg)
	}
	snapshot := history.snapshot(cur, now)
	assert.Equal(t, cur.Text, snapshot.current.text.value)
	require.Len(t, snapshot.previous, 2)
	assert.Equal(t, "boundary", snapshot.previous[0].text.value)
	assert.Equal(t, "recent", snapshot.previous[1].text.value)
	assert.Len(t, history.messages, 7)

	cur.ID, cur.Text = 11, "unrecorded request"
	snapshot = history.snapshot(cur, now)
	assert.Equal(t, cur.Text, snapshot.current.text.value)
	require.Len(t, snapshot.previous, 3)
	assert.Equal(t, "current request", snapshot.previous[2].text.value)
	var empty LimitedMessageHistory
	assert.Equal(t, cur.Text, empty.snapshot(cur, now).current.text.value)
}

func TestLimitedMessageHistory_snapshot_Bounds(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	history := NewLimitedMessageHistory(20)
	for i := 1; i <= 10; i++ {
		history.Add(bot.Message{ID: i, ChatID: -100, Sent: now.Add(-time.Minute), Text: fmt.Sprint(i) + strings.Repeat("я", 1500)})
	}
	cur := bot.Message{ID: 11, ChatID: -100, Sent: now, Text: strings.Repeat("ж", 1490) + " question?"}
	snapshot := history.snapshot(cur, now)
	assert.Equal(t, cur.Text, snapshot.current.text.value)
	require.Len(t, snapshot.previous, 6)
	assert.True(t, strings.HasPrefix(snapshot.previous[0].text.value, "5"))
	for _, entry := range snapshot.previous {
		assert.Equal(t, 1000, utf8.RuneCountInString(entry.text.value))
		assert.True(t, utf8.ValidString(entry.text.value))
		assert.True(t, entry.text.truncated)
	}
	assert.Greater(t, utf8.RuneCountInString(history.messages[0].Text), 1000)

	cur.Text = strings.Repeat("ж", 7800)
	cur.ReplyTo.From = bot.User{Username: "parent"}
	cur.ReplyTo.Text = strings.Repeat("я", 1200)
	snapshot = history.snapshot(cur, now)
	assert.Empty(t, snapshot.previous)
	assert.Equal(t, cur.Text, snapshot.current.text.value)
	require.NotNil(t, snapshot.current.reply)
	assert.Equal(t, 200, utf8.RuneCountInString(snapshot.current.reply.text.value))
	assert.True(t, snapshot.current.reply.text.truncated)

	cur.Text = strings.Repeat("ж", 8000)
	snapshot = history.snapshot(cur, now)
	assert.Empty(t, snapshot.current.reply.text.value)
	assert.True(t, snapshot.current.reply.text.truncated)
}

func TestHistorySnapshot_chatMessages(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	history := NewLimitedMessageHistory(10)
	first := bot.Message{ID: 1, ChatID: -100, Sent: now.Add(-time.Minute), Text: "context", From: bot.User{ID: 99111999, Username: "alice"}}
	first.ReplyTo.From = bot.User{ID: 88222888, DisplayName: "Bob"}
	first.ReplyTo.Text = strings.Repeat("quoted", 200)
	history.Add(first)
	imageMessage := bot.Message{ID: 2, ChatID: -100, Sent: now.Add(-time.Second), From: bot.User{ID: 77333777}, Image: &bot.Image{Caption: "image caption"}}
	history.Add(imageMessage)
	cur := bot.Message{ID: 3, ChatID: -100, Sent: now, Text: "chat! original", From: imageMessage.From}
	cur.ReplyTo.From = bot.User{ID: 66444666}
	snapshot := history.snapshot(cur, now)
	messages := snapshot.chatMessages("system prompt", "custom prompt.\nstripped request")
	require.Len(t, messages, 4)
	assert.Equal(t, ai.ChatMessageRoleSystem, messages[0].Role)
	assert.Equal(t, "system prompt", messages[0].Content)
	assert.Contains(t, messages[1].Content, "@alice: context")
	assert.Contains(t, messages[1].Content, "in reply to Bob:")
	assert.Contains(t, messages[1].Content, "[truncated]")
	assert.Contains(t, messages[1].Content, first.Sent.Format(time.RFC3339))
	assert.Contains(t, messages[2].Content, "image caption")
	assert.Contains(t, messages[2].Content, "[image unavailable]")
	assert.Equal(t, snapshot.previous[1].author, snapshot.current.author)
	assert.NotEqual(t, snapshot.current.author, snapshot.current.reply.author)
	assert.Contains(t, messages[3].Content, snapshot.current.author+": custom prompt.\nstripped request")
	assert.Contains(t, messages[3].Content, "[unavailable]")
	for _, message := range messages[1:] {
		assert.Equal(t, ai.ChatMessageRoleUser, message.Role)
		for _, secret := range []string{"99111999", "88222888", "77333777", "66444666", "chat! original"} {
			assert.NotContains(t, message.Content, secret)
		}
	}
	assert.Equal(t, "chat! original", snapshot.current.text.value)
}

func TestLimitedMessageHistory_snapshot_AuthorIdentity(t *testing.T) {
	now := time.Now()
	history := NewLimitedMessageHistory(10)
	for _, msg := range []bot.Message{
		{ID: 1, ChatID: -100, Sent: now, From: bot.User{ID: 1, DisplayName: "Alex"}},
		{ID: 2, ChatID: -100, Sent: now, From: bot.User{ID: 2, DisplayName: "Alex"}},
		{ID: 3, ChatID: -100, Sent: now},
		{ID: 4, ChatID: -100, Sent: now},
		{ID: 5, ChatID: -100, Sent: now, SenderChat: bot.SenderChat{ID: -500, UserName: "channel"}},
	} {
		history.Add(msg)
	}
	cur := bot.Message{ID: 6, ChatID: -100, Sent: now, From: bot.User{ID: 1, DisplayName: "renamed"}}
	snapshot := history.snapshot(cur, now)
	require.Len(t, snapshot.previous, 5)
	assert.Equal(t, "Alex", snapshot.previous[0].author)
	assert.NotEqual(t, snapshot.previous[0].author, snapshot.previous[1].author)
	assert.NotEqual(t, snapshot.previous[2].author, snapshot.previous[3].author)
	assert.Equal(t, "@channel", snapshot.previous[4].author)
	assert.Equal(t, snapshot.previous[0].author, snapshot.current.author)
}
