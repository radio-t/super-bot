package bot

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestFilter_OnMessage(t *testing.T) {
	superUser := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}
	mockedHTTPClient := &mocks.HTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.String(), "101") {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(`{"ok": true, "description": "Is a spammer"}`)),
				}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok": false, "description": "Not a spammer"}`)),
			}, nil
		},
	}

	spamSamples := strings.NewReader("win free iPhone\nlottery prize")

	filter := NewSpamFilter(SpamParams{
		SpamSamples:         spamSamples,
		SimilarityThreshold: 0.5,
		SuperUser:           superUser,
		MinMsgLen:           5,
		Dry:                 false,
		HTTPClient:          mockedHTTPClient,
	})
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
				BanInterval: permanentBanDuration, ReplyTo: 4, DeleteReplyTo: true,
				User: User{ID: 4, Username: "john", DisplayName: "John"}},
		},
		{
			Message{From: User{ID: 2, Username: "spammer", DisplayName: "Spammer"}, Text: "Win a free iPhone now!", ID: 2},
			Response{Text: "this is spam! go to ban, \"Spammer\" (id:2)", Send: true,
				ReplyTo: 2, BanInterval: permanentBanDuration, DeleteReplyTo: true,
				User: User{ID: 2, Username: "spammer", DisplayName: "Spammer"},
			},
		},
		{
			Message{From: User{ID: 3, Username: "super", DisplayName: "SuperUser"}, Text: "Win a free iPhone now!", ID: 3},
			Response{},
		},
		{
			Message{From: User{ID: 101, Username: "spammer", DisplayName: "blah"}, Text: "something something", ID: 10},
			Response{Text: "this is spam! go to ban, \"blah\" (id:101)", Send: true,
				ReplyTo: 10, BanInterval: permanentBanDuration, DeleteReplyTo: true,
				User: User{ID: 101, Username: "spammer", DisplayName: "blah"},
			},
		},
		{
			Message{From: User{ID: 102, Username: "spammer", DisplayName: "blah"}, Text: "something пишите в лс something", ID: 10},
			Response{Text: "this is spam! go to ban, \"blah\" (id:102)", Send: true,
				ReplyTo: 10, BanInterval: permanentBanDuration, DeleteReplyTo: true,
				User: User{ID: 102, Username: "spammer", DisplayName: "blah"},
			},
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
	filter := NewSpamFilter(SpamParams{
		SpamSamples:         spamSamples,
		SimilarityThreshold: 0.5,
		SuperUser:           nil,
		MinMsgLen:           5,
		Dry:                 false,
	})

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
			filter.SimilarityThreshold = test.threshold // update threshold for each test case
			assert.Equal(t, test.expected, filter.isSpamSimilarity(test.message))
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
	filter := NewSpamFilter(SpamParams{
		SpamSamples:         spamSamples,
		SimilarityThreshold: 0.5,
		SuperUser:           nil,
		MinMsgLen:           5,
		Dry:                 false,
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isSpam, count := filter.tooManyEmojis(tt.input, 2)
			assert.Equal(t, tt.count, count)
			assert.Equal(t, tt.spam, isSpam)
		})
	}
}

func TestStopWords(t *testing.T) {
	filter := &SpamFilter{}

	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "Stop word present",
			message:  "Hello, please send me a message в личку.",
			expected: true,
		},
		{
			name:     "Stop word present with emoji",
			message:  "👋Всем привет\nИщу амбициозного человека к се6е в команду\nКто в поисках дополнительного заработка или хочет попробовать себя в новой  сфере деятельности! 👨🏻\u200d💻\nПишите в лс✍️",
			expected: true,
		},
		{
			name:     "Stop word present with emoji, 2",
			message:  "Ищy людeй для coвмecтнoгo зaрaбoткa нa цифрoвыx aктивax. 🪙\nДнeвнaя дoxoднocть cocтaвляeт 3-6%.💸\nРaбoтaeм в удaлeннoм фopмaтe. 🌐\nГoтoв coтpyдничaть c нoвичкaми. 🤝\nНe тpeбyeтся внeceниe пpeдвapитeльныx плaтeжeй, пpинимaю пpoцeнт c пoлyченнoй прибыли. 🙌💰\nЗaинтepecoвaлo?\nПишитe в личныe cooбщeния. 📝",
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

func TestSpam_isCasSpam(t *testing.T) {

	tests := []struct {
		name           string
		mockResp       string
		mockStatusCode int
		expected       bool
	}{
		{
			name:           "User is not a spammer",
			mockResp:       `{"ok": false, "description": "Not a spammer"}`,
			mockStatusCode: 200,
			expected:       false,
		},
		{
			name:           "User is a spammer",
			mockResp:       `{"ok": true, "description": "Is a spammer"}`,
			mockStatusCode: 200,
			expected:       true,
		},
		{
			name:           "HTTP error",
			mockResp:       "",
			mockStatusCode: 500,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockedHTTPClient := &mocks.HTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.mockStatusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.mockResp)),
					}, nil
				},
			}

			s := NewSpamFilter(SpamParams{
				CasAPI:      "http://localhost",
				HTTPClient:  mockedHTTPClient,
				SpamSamples: strings.NewReader("win free iPhone\nlottery prize"),
			})

			msg := Message{
				From: User{
					ID:          1,
					Username:    "testuser",
					DisplayName: "Test User",
				},
				ID:   1,
				Text: "Hello",
			}

			isSpam := s.isCasSpam(msg.From.ID)
			assert.Equal(t, tt.expected, isSpam)
		})
	}
}

func TestSpam_OnMessageCheckOnce(t *testing.T) {
	mockedHTTPClient := &mocks.HTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok": false, "description": "Not a spammer"}`)),
			}, nil
		},
	}

	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}

	s := NewSpamFilter(SpamParams{
		CasAPI:              "http://localhost",
		HTTPClient:          mockedHTTPClient,
		SpamSamples:         strings.NewReader("win free iPhone\nlottery prize"),
		SuperUser:           su,
		SimilarityThreshold: 0.5,
	})
	res := s.OnMessage(Message{From: User{ID: 1, Username: "testuser"}, ID: 1, Text: "Hello"})
	assert.Equal(t, Response{}, res)
	assert.Len(t, mockedHTTPClient.DoCalls(), 1, "Do should be called once")

	res = s.OnMessage(Message{From: User{ID: 1, Username: "testuser"}, ID: 1, Text: "Hello"})
	assert.Equal(t, Response{}, res)
	assert.Len(t, mockedHTTPClient.DoCalls(), 1, "Do should not be called anymore")

	res = s.OnMessage(Message{From: User{ID: 2, Username: "testuser2"}, ID: 2, Text: "Hello"})
	assert.Equal(t, Response{}, res)
	assert.Len(t, mockedHTTPClient.DoCalls(), 2, "Do should be called once more")
}
