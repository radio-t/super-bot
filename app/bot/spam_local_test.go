package bot

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestSpamLocalFilter_OnMessage(t *testing.T) {
	superUser := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}
	spamSamples := strings.NewReader("win free iPhone\nlottery prize")

	filter := NewSpamLocalFilter(spamSamples, 0.5, superUser, 5, false)
	tests := []struct {
		msg      Message
		expected Response
	}{
		{
			Message{From: User{ID: 1, Username: "john", DisplayName: "John"}, Text: "Hello, how are you?", ID: 1},
			Response{},
		},
		{
			Message{From: User{ID: 4, Username: "john", DisplayName: "John"}, Text: "Hello 😁🐶🍕 how are you? ", ID: 4},
			Response{Text: "this is spam! go to ban, \"John\" (id:4)", Send: true,
				BanInterval: permanentBanDuration, ReplyTo: 4, DeleteReplyTo: true},
		},
		{
			Message{From: User{ID: 2, Username: "spammer", DisplayName: "Spammer"}, Text: "Win a free iPhone now!", ID: 2},
			Response{Text: "this is spam! go to ban, \"Spammer\" (id:2)", Send: true, ReplyTo: 2, BanInterval: permanentBanDuration, DeleteReplyTo: true},
		},
		{
			Message{From: User{ID: 3, Username: "super", DisplayName: "SuperUser"}, Text: "Win a free iPhone now!", ID: 3},
			Response{},
		},
	}

	for _, test := range tests {
		t.Run(test.msg.From.Username, func(t *testing.T) {
			assert.Equal(t, test.expected, filter.OnMessage(test.msg))
		})
	}
}

func TestIsSpam(t *testing.T) {
	spamSamples := strings.NewReader("win free iPhone\nlottery prize")
	filter := NewSpamLocalFilter(spamSamples, 0.5, nil, 5, false) // SuperUser set to nil for this test

	tests := []struct {
		name      string
		message   string
		threshold float64
		expected  bool
	}{
		{"Not Spam", "Hello, how are you?", 0.5, false},
		{"Exact Match", "Win a free iPhone now!", 0.5, true},
		{"Similar Match", "You won a lottery prize!", 0.3, true},
		{"High Threshold", "You won a lottery prize!", 0.9, false},
		{"Partial Match", "win free", 0.9, false},
		{"Low Threshold", "win free", 0.8, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter.threshold = test.threshold // Update threshold for each test case
			assert.Equal(t, test.expected, filter.isSpam(test.message))
		})
	}
}

// nolint
func TestTooManyEmojis(t *testing.T) {
	tests := []struct {
		name  string
		input string
		count int
		spam  bool
	}{
		{"NoEmoji", "Hello, world!", 0, false},
		{"OneEmoji", "Hi there 👋", 1, false},
		{"TwoEmojis", "Good morning 🌞🌻", 2, false},
		{"Mixed", "👨‍👩‍👧‍👦 Family emoji", 1, false},
		{"EmojiSequences", "🏳️‍🌈 Rainbow flag", 1, false},
		{"TextAfterEmoji", "😊 Have a nice day!", 1, false},
		{"OnlyEmojis", "😁🐶🍕", 3, true},
		{"WithCyrillic", "Привет 🌞 🍕 мир! 👋", 3, true},
	}

	spamSamples := strings.NewReader("win free iPhone\nlottery prize")
	filter := NewSpamLocalFilter(spamSamples, 0.5, nil, 5, false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSpam, count := filter.tooManyEmojis(tt.input, 2)
			assert.Equal(t, tt.count, count)
			assert.Equal(t, tt.spam, isSpam)
		})
	}
}

func TestStopWords(t *testing.T) {
	filter := &SpamLocalFilter{}

	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "Stop word present",
			message:  "Hello, please send me a message в личку",
			expected: true,
		},
		{
			name:     "No stop word present",
			message:  "Hello, how are you?",
			expected: false,
		},
		{
			name:     "Case insensitive stop word present",
			message:  "Hello, please send me a message В ЛИЧКУ",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, filter.stopWords(test.message))
		})
	}
}
