package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"testing"
	"time"

	ai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot"
	bmocks "github.com/radio-t/super-bot/app/bot/mocks"
	"github.com/radio-t/super-bot/app/bot/openai/jev"
	"github.com/radio-t/super-bot/app/bot/openai/mocks"
)

func testGateResponse(invites, answerable, spam float64) jev.Response {
	return jev.Response{Model: "jev-test", Answers: map[string]jev.Answer{
		"invites": {Type: "noul", Noul: &invites}, "answerable": {Type: "noul", Noul: &answerable}, "spam": {Type: "noul", Noul: &spam},
	}}
}

func newGateTestBot() (*OpenAI, *mocks.JevClient, *mocks.OpenAIClient) {
	gate := &mocks.JevClient{AskFunc: func(context.Context, any, map[string]jev.Question) (jev.Response, error) {
		return testGateResponse(1, 1, 0), nil
	}}
	client := &mocks.OpenAIClient{CreateChatCompletionFunc: func(context.Context, ai.ChatCompletionRequest) (ai.ChatCompletionResponse, error) {
		return ai.ChatCompletionResponse{Choices: []ai.ChatCompletionChoice{{Message: ai.ChatCompletionMessage{Content: "A compiler translates code."}}}}, nil
	}}
	params := getDefaultTestingConfig()
	params.Jev = gate
	su := &bmocks.SuperUser{IsSuperFunc: func(string) bool { return false }}
	o := NewOpenAI(params, &http.Client{Timeout: time.Second}, su)
	o.client = client
	o.nowFn = func() time.Time { return time.Date(2026, 9, 22, 12, 10, 0, 0, time.UTC) }
	return o, gate, client
}

func TestOpenAI_AutoReplyGate(t *testing.T) {
	o, gate, client := newGateTestBot()
	msg := bot.Message{ID: 10, ChatID: -100, Sent: o.nowFn(), Text: "Please explain how a compiler works", From: bot.User{ID: 12, Username: "asker"}}

	response := o.OnMessage(msg)
	require.True(t, response.Send)
	assert.Equal(t, msg.ID, response.ReplyTo)
	assert.Zero(t, response.BanInterval)
	assert.True(t, o.lastDT.IsZero())
	require.Len(t, gate.AskCalls(), 1)
	require.Len(t, client.CreateChatCompletionCalls(), 1)
	request := client.CreateChatCompletionCalls()[0].ChatCompletionRequest
	assert.Contains(t, request.Messages[0].Content, "50 words")
	assert.NotContains(t, request.Messages[0].Content, "substantive remark")
	assert.Contains(t, request.Messages[len(request.Messages)-1].Content, "@asker: "+msg.Text)

	assert.False(t, o.OnMessage(msg).Send)
	assert.Len(t, gate.AskCalls(), 1)
	assert.Len(t, client.CreateChatCompletionCalls(), 1)
}

func TestHistorySnapshot_jevState(t *testing.T) {
	o, _, _ := newGateTestBot()
	now := o.nowFn()
	parent := bot.Message{ID: 1, ChatID: -100, Sent: now.Add(-time.Minute), From: bot.User{ID: 88222888},
		Image: &bot.Image{Caption: strings.Repeat("x", 1200)}}
	parent.ReplyTo.From = bot.User{Username: "earlier"}
	parent.ReplyTo.Text = "earlier technical detail"
	o.history.Add(parent)
	cur := bot.Message{ID: 2, ChatID: -100, Sent: now, From: parent.From, Text: "Please explain this topic"}
	cur.ReplyTo.From = bot.User{ID: 99111999, Username: "parent"}
	snapshot := o.history.snapshot(cur, now)
	state := snapshot.jevState()
	assert.Equal(t, snapshot.current.author, state.Message.Author)
	assert.Equal(t, cur.Text, state.Message.Text)
	require.Len(t, state.History, 1)
	assert.Equal(t, state.Message.Author, state.History[0].Author)
	assert.True(t, state.History[0].Image)
	assert.True(t, state.History[0].Truncated)
	assert.Equal(t, parent.Sent.Format(time.RFC3339), state.History[0].Time)
	require.NotNil(t, state.History[0].ReplyTo)
	assert.Equal(t, "earlier technical detail", state.History[0].ReplyTo.Text)
	assert.Equal(t, "@earlier", state.History[0].ReplyTo.Author)
	require.NotNil(t, state.Message.ReplyTo)
	assert.Equal(t, "@parent", state.Message.ReplyTo.Author)
	assert.True(t, state.Message.ReplyTo.Unavailable)
	data, err := json.Marshal(state)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "88222888")
	assert.NotContains(t, string(data), "99111999")
	assert.Contains(t, string(data), `"message"`)
	assert.Contains(t, string(data), `"reply_to"`)
	assert.Contains(t, snapshot.chatMessages("system", cur.Text)[2].Content, state.Message.Author+": "+state.Message.Text)

	cur.Text = strings.Repeat("я", 8001)
	snapshot = o.history.snapshot(cur, now)
	assert.True(t, snapshot.oversized)
	assert.Equal(t, cur.Text, snapshot.jevState().Message.Text)
	cur.Text = strings.Repeat("я", 8000)
	assert.False(t, o.history.snapshot(cur, now).oversized)
}

func TestOpenAI_shouldJoin(t *testing.T) {
	tests := []struct {
		name       string
		invites    float64
		answerable float64
		spam       float64
		want       bool
	}{
		{"at positive boundaries", 0.8, 0.7, 0.299, true},
		{"below invites", 0.799, 1, 0, false},
		{"above invites", 0.801, 1, 0, true},
		{"below answerable", 1, 0.699, 0, false},
		{"above answerable", 1, 0.701, 0, true},
		{"spam boundary veto", 1, 1, 0.3, false},
		{"spam above boundary", 1, 1, 0.301, false},
		{"invalid invites", 1.01, 1, 0, false},
		{"invalid spam", 1, 1, -0.01, false},
		{"NaN", math.NaN(), 1, 0, false},
		{"infinity", 1, math.Inf(1), 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o, gate, _ := newGateTestBot()
			gate.AskFunc = func(context.Context, any, map[string]jev.Question) (jev.Response, error) {
				return testGateResponse(tt.invites, tt.answerable, tt.spam), nil
			}
			snapshot := o.history.snapshot(bot.Message{Text: "Please explain compilers"}, o.nowFn())
			assert.Equal(t, tt.want, o.shouldJoin(snapshot))
			require.Len(t, gate.AskCalls(), 1)
			questions := gate.AskCalls()[0].Questions
			require.Len(t, questions, 3)
			for _, key := range []string{"invites", "answerable", "spam"} {
				assert.Equal(t, "noul", questions[key].Type)
				assert.NotEmpty(t, questions[key].Instructions)
			}
		})
	}
}

func TestOpenAI_shouldJoin_Invalid(t *testing.T) {
	for _, invalid := range []string{"error", "missing answer", "null noul", "wrong type", "missing model", "nil client", "oversized"} {
		t.Run(invalid, func(t *testing.T) {
			o, gate, _ := newGateTestBot()
			response := testGateResponse(1, 1, 0)
			var err error
			snapshot := o.history.snapshot(bot.Message{Text: "Please explain compilers"}, o.nowFn())
			switch invalid {
			case "error":
				err = context.DeadlineExceeded
			case "missing answer":
				delete(response.Answers, "spam")
			case "null noul":
				response.Answers["spam"] = jev.Answer{Type: "noul"}
			case "wrong type":
				value := 0.0
				response.Answers["spam"] = jev.Answer{Type: "choice", Noul: &value}
			case "missing model":
				response.Model = ""
			case "nil client":
				o.params.Jev = nil
			case "oversized":
				snapshot = o.history.snapshot(bot.Message{Text: strings.Repeat("x", 8001)}, o.nowFn())
			}
			gate.AskFunc = func(context.Context, any, map[string]jev.Question) (jev.Response, error) { return response, err }
			assert.False(t, o.shouldJoin(snapshot))
			if invalid == "nil client" || invalid == "oversized" {
				assert.Empty(t, gate.AskCalls())
			} else {
				assert.Len(t, gate.AskCalls(), 1)
			}
		})
	}
}

func TestAutoReplyLimits(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 10, 0, 0, time.FixedZone("local", 5*3600+30*60))
	t.Run("cooldown", func(t *testing.T) {
		var limits autoReplyLimits
		require.True(t, limits.allowed(now))
		limits.mark(now)
		assert.False(t, limits.allowed(now.Add(15*time.Minute-time.Nanosecond)))
		assert.True(t, limits.allowed(now.Add(15*time.Minute)))
	})
	t.Run("daily cap", func(t *testing.T) {
		var limits autoReplyLimits
		for i := range 10 {
			at := now.Add(time.Duration(i) * 15 * time.Minute)
			require.True(t, limits.allowed(at))
			limits.mark(at)
		}
		assert.False(t, limits.allowed(now.Add(3*time.Hour)))
		assert.True(t, limits.allowed(now.AddDate(0, 0, 1)))
	})
	t.Run("midnight preserves cooldown", func(t *testing.T) {
		var limits autoReplyLimits
		late := time.Date(2026, 9, 22, 23, 50, 0, 0, now.Location())
		for range 10 {
			limits.mark(late)
		}
		assert.False(t, limits.allowed(late.Add(10*time.Minute)))
		assert.True(t, limits.allowed(late.Add(15*time.Minute)))
	})
	t.Run("clock hour budget", func(t *testing.T) {
		var limits autoReplyLimits
		for range 150 {
			require.True(t, limits.allowed(now))
			limits.noteJevCall(now)
		}
		assert.False(t, limits.allowed(now))
		assert.False(t, limits.allowed(now.Add(50*time.Minute-time.Nanosecond)))
		assert.True(t, limits.allowed(now.Add(50*time.Minute)))
	})
}

func TestOpenAI_AutoReply_Ineligible(t *testing.T) {
	for _, reason := range []string{"disabled", "nil client", "private", "idle", "short", "oversized"} {
		t.Run(reason, func(t *testing.T) {
			o, gate, client := newGateTestBot()
			msg := bot.Message{ID: 1, ChatID: -100, Sent: o.nowFn(), Text: "Please explain how compilers work"}
			switch reason {
			case "disabled":
				o.params.EnableAutoResponse = false
			case "nil client":
				o.params.Jev = nil
			case "private":
				msg.ChatID = 100
			case "idle":
				msg.ChatID, msg.Text = 0, "idle"
			case "short":
				msg.Text = "hi"
			case "oversized":
				msg.Text = strings.Repeat("я", 8001)
			}
			assert.False(t, o.OnMessage(msg).Send)
			assert.Empty(t, gate.AskCalls())
			assert.Empty(t, client.CreateChatCompletionCalls())
		})
	}
}

func TestOpenAI_AutoReply_HourlyBudget(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprintf("error_%t", failed), func(t *testing.T) {
			o, gate, client := newGateTestBot()
			now := o.nowFn()
			o.nowFn = func() time.Time { return now }
			gate.AskFunc = func(context.Context, any, map[string]jev.Question) (jev.Response, error) {
				if failed {
					return jev.Response{}, context.DeadlineExceeded
				}
				return testGateResponse(0, 1, 0), nil
			}
			for i := range 160 {
				assert.False(t, o.OnMessage(bot.Message{ID: i + 1, ChatID: -100, Sent: now, Text: "Please explain this topic"}).Send)
			}
			assert.Len(t, gate.AskCalls(), 150)
			assert.Empty(t, client.CreateChatCompletionCalls())
			now = now.Add(50 * time.Minute)
			assert.False(t, o.OnMessage(bot.Message{ID: 200, ChatID: -100, Sent: now, Text: "Please explain this topic"}).Send)
			assert.Len(t, gate.AskCalls(), 151)
		})
	}
}

func TestOpenAI_AutoReply_LimitsWithBotReply(t *testing.T) {
	o, gate, client := newGateTestBot()
	now := o.nowFn()
	o.nowFn = func() time.Time { return now }
	msg := bot.Message{ID: 1, ChatID: -100, Text: "Please explain this topic"}
	msg.ReplyTo.From = bot.User{ID: 99, Username: "the_bot"}
	msg.ReplyTo.Text = "Previous bot answer"
	for i := range 10 {
		msg.ID, msg.Sent = i+1, now
		require.True(t, o.OnMessage(msg).Send)
		assert.False(t, o.OnMessage(msg).Send)
		now = now.Add(15 * time.Minute)
	}
	assert.False(t, o.OnMessage(msg).Send)
	assert.Len(t, gate.AskCalls(), 10)
	assert.Len(t, client.CreateChatCompletionCalls(), 10)
	assert.True(t, o.lastDT.IsZero())
}

func TestOpenAI_AutoReply_NoOutput(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprintf("error_%t", failed), func(t *testing.T) {
			o, gate, client := newGateTestBot()
			client.CreateChatCompletionFunc = func(context.Context, ai.ChatCompletionRequest) (ai.ChatCompletionResponse, error) {
				if failed {
					return ai.ChatCompletionResponse{}, fmt.Errorf("upstream unavailable")
				}
				return ai.ChatCompletionResponse{Choices: []ai.ChatCompletionChoice{{Message: ai.ChatCompletionMessage{Content: " \n"}}}}, nil
			}
			for i := range 2 {
				assert.False(t, o.OnMessage(bot.Message{ID: i + 1, ChatID: -100, Sent: o.nowFn(), Text: "Please explain this topic"}).Send)
			}
			assert.Len(t, gate.AskCalls(), 2)
			assert.Len(t, client.CreateChatCompletionCalls(), 2)
			assert.True(t, o.lastDT.IsZero())
		})
	}
}

func TestOpenAI_DirectBypassesAutoLimits(t *testing.T) {
	o, gate, client := newGateTestBot()
	for range 150 {
		o.autoLimits.noteJevCall(o.nowFn())
	}
	for range 10 {
		o.autoLimits.mark(o.nowFn())
	}
	msg := bot.Message{ID: 1, ChatID: 100, Sent: o.nowFn(), Text: "chat! Explain compilers"}
	require.True(t, o.OnMessage(msg).Send)
	assert.Empty(t, gate.AskCalls())
	assert.Len(t, client.CreateChatCompletionCalls(), 1)
	assert.Equal(t, o.nowFn(), o.lastDT)
}

func TestOpenAI_PrivateDirectSharesCooldown(t *testing.T) {
	o, gate, client := newGateTestBot()
	msg := bot.Message{ID: 1, ChatID: -100, Sent: o.nowFn(), Text: "chat! Explain compilers"}
	require.True(t, o.OnMessage(msg).Send)
	msg.ID, msg.ChatID = 2, 100
	response := o.OnMessage(msg)
	assert.True(t, response.Send)
	assert.Equal(t, time.Hour, response.BanInterval)
	assert.Equal(t, 2, response.ReplyTo)
	assert.Len(t, client.CreateChatCompletionCalls(), 1)
	assert.Empty(t, gate.AskCalls())
}
