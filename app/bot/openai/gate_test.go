package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"testing/iotest"
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

func syntheticReplayCase(id int, at time.Time) gateReplayCase {
	return gateReplayCase{Version: 1, CaseID: fmt.Sprintf("case-%d", id), ConversationID: fmt.Sprintf("thread-%d", id),
		ChatID: -100, At: at, HistorySize: 10, Context: []bot.Message{},
		Current: bot.Message{ID: id, ChatID: -100, Sent: at, Text: "Please explain how a compiler works", From: bot.User{ID: 99111999, Username: "asker"}},
		Stratum: "random", Split: "dev", Label: "yes", Reason: "A self-contained request", EvidenceNeeded: []string{"none"}}
}

func TestLoadGateReplayCases(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	valid := syntheticReplayCase(10, now)
	for _, change := range []string{"valid", "version", "missing context", "chat mismatch", "private", "time mismatch", "future context", "self context", "future parent", "duplicate context", "unsorted context", "too much context", "bad size", "bad label", "bad stratum", "bad split", "missing reason", "bad evidence", "source gap", "source truncation", "direct command", "duplicate case", "duplicate current", "conversation split", "day split", "context split"} {
		t.Run(change, func(t *testing.T) {
			item := valid
			second := syntheticReplayCase(20, now.Add(24*time.Hour))
			records := []gateReplayCase{}
			switch change {
			case "version":
				item.Version = 2
			case "missing context":
				item.Context = nil
			case "chat mismatch":
				item.ChatID = -200
			case "private":
				item.ChatID, item.Current.ChatID = 100, 100
			case "time mismatch":
				item.At = now.Add(time.Second)
			case "future context":
				item.Context = []bot.Message{{ID: 1, ChatID: -100, Sent: now.Add(time.Second)}}
			case "self context":
				item.Context = []bot.Message{item.Current}
			case "future parent":
				item.Current.ReplyTo.Sent = now.Add(time.Second)
			case "duplicate context":
				msg := bot.Message{ID: 1, ChatID: -100, Sent: now.Add(-time.Minute)}
				item.Context = []bot.Message{msg, msg}
			case "unsorted context":
				item.Context = []bot.Message{{ID: 1, ChatID: -100, Sent: now.Add(-time.Minute)}, {ID: 2, ChatID: -100, Sent: now.Add(-2 * time.Minute)}}
			case "too much context":
				item.HistorySize = 1
				item.Context = []bot.Message{{ID: 1}, {ID: 2}}
			case "bad size":
				item.HistorySize = 0
			case "bad label":
				item.Label = "positive"
			case "bad stratum":
				item.Stratum = "baseline"
			case "bad split":
				item.Split = "test"
			case "missing reason":
				item.Reason = ""
			case "bad evidence":
				item.EvidenceNeeded = []string{"none", "history"}
			case "source gap":
				item.SourceMissing = []string{"missing incoming message"}
			case "source truncation":
				item.SourceTruncated = true
			case "direct command":
				item.Current.Text = "chat! explain compilers"
			case "duplicate case":
				second.CaseID = item.CaseID
				records = append(records, second)
			case "duplicate current":
				second.Current = item.Current
				second.ChatID, second.At = item.ChatID, item.At
				records = append(records, second)
			case "conversation split":
				second.ConversationID, second.Split = item.ConversationID, "holdout"
				records = append(records, second)
			case "day split":
				second.At, second.Current.Sent, second.Split = now.Add(time.Hour), now.Add(time.Hour), "holdout"
				records = append(records, second)
			case "context split":
				item.At = time.Date(2026, 9, 22, 23, 55, 0, 0, time.UTC)
				item.Current.Sent = item.At
				second.At, second.Current.Sent, second.Split = item.At.Add(10*time.Minute), item.At.Add(10*time.Minute), "holdout"
				second.Context = []bot.Message{item.Current}
				records = append(records, second)
			}
			records = append(records, item)
			var input bytes.Buffer
			input.WriteString("\n")
			for _, record := range records {
				require.NoError(t, json.NewEncoder(&input).Encode(record))
			}
			loaded, err := loadGateReplayCases(&input)
			if change != "valid" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Len(t, loaded, 1)
			assert.Equal(t, valid, loaded[0])
		})
	}
}

func TestLoadGateReplayCases_BadInput(t *testing.T) {
	for _, input := range []string{"", "{}", "{", "{} {}", `{"unexpected":true}`, strings.Repeat("x", 1024*1024+1)} {
		_, err := loadGateReplayCases(strings.NewReader(input))
		require.Error(t, err)
	}
	_, err := loadGateReplayCases(iotest.ErrReader(io.ErrUnexpectedEOF))
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestLoadGateReplayCases_UnjudgeableAndExpiredContext(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	first := syntheticReplayCase(10, now)
	first.Label, first.SourceTruncated = "unjudgeable", true
	first.SourceMissing = []string{"known gap in incoming log"}
	first.EvidenceNeeded = []string{"unknown"}
	second := syntheticReplayCase(20, now.Add(24*time.Hour))
	second.Split, second.Stratum = "holdout", "past_reply"
	second.Context = []bot.Message{first.Current}
	var input bytes.Buffer
	for _, item := range []gateReplayCase{first, second} {
		require.NoError(t, json.NewEncoder(&input).Encode(item))
	}
	cases, err := loadGateReplayCases(&input)
	require.NoError(t, err)
	require.Len(t, cases, 2)
	assert.Equal(t, first, cases[0])
	assert.Empty(t, cases[1].blindView().State.History)
}

func TestGateReplay_ProductionPipeline(t *testing.T) {
	o, client, openaiClient := newGateTestBot()
	item := syntheticReplayCase(10, o.nowFn())
	item.HistorySize = 2
	item.Context = []bot.Message{
		{ID: 1, ChatID: -100, Sent: item.At.Add(-2 * time.Minute), Text: "evicted-before-snapshot"},
		{ID: 2, ChatID: -100, Sent: item.At.Add(-time.Minute), Text: strings.Repeat("я", 1200)},
	}
	item.Current.ReplyTo.From = bot.User{Username: "parent"}
	item.Current.ReplyTo.Text = "quoted context"
	replay := gateReplay{client: client}
	result, err := replay.run(item)
	require.NoError(t, err)
	assert.True(t, result.Join)
	assert.Empty(t, result.Error)
	assert.Equal(t, "jev-test", result.Model)
	assert.InDelta(t, 1, *result.Invites, 0.0001)
	require.Len(t, client.AskCalls(), 1)
	state := client.AskCalls()[0].State.(jevState)
	require.Len(t, state.History, 1)
	assert.True(t, state.History[0].Truncated)
	assert.Equal(t, item.Current.Text, state.Message.Text)
	assert.Equal(t, "quoted context", state.Message.ReplyTo.Text)
	assert.Empty(t, openaiClient.CreateChatCompletionCalls())

	view := item.blindView()
	data, err := json.Marshal(view)
	require.NoError(t, err)
	for _, hidden := range []string{`"label"`, `"stratum"`, `"split"`, `"reason"`, `"evidence_needed"`, "99111999", "evicted-before-snapshot"} {
		assert.NotContains(t, string(data), hidden)
	}
	assert.Equal(t, state, view.State)

	client.AskFunc = func(context.Context, any, map[string]jev.Question) (jev.Response, error) {
		return testGateResponse(0.799, 1, 0), nil
	}
	result, err = replay.run(item)
	require.NoError(t, err)
	assert.False(t, result.Join)
	assert.Empty(t, result.Error)
	client.AskFunc = func(context.Context, any, map[string]jev.Question) (jev.Response, error) {
		return jev.Response{}, context.DeadlineExceeded
	}
	result, err = replay.run(item)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Error)
	client.AskFunc = func(context.Context, any, map[string]jev.Question) (jev.Response, error) { return jev.Response{}, nil }
	result, err = replay.run(item)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Error)

	item.Current.Text = strings.Repeat("x", 8001)
	before := len(client.AskCalls())
	result, err = replay.run(item)
	require.NoError(t, err)
	assert.False(t, result.Join)
	assert.Equal(t, "oversized", result.Skipped)
	assert.Empty(t, result.Error)
	assert.Len(t, client.AskCalls(), before)
}

func TestGateReplayMetrics(t *testing.T) {
	stats := map[string]*gateReplayStats{}
	for _, result := range []gateReplayResult{
		{Split: "dev", Stratum: "random", Label: "yes", Join: true},
		{Split: "dev", Stratum: "random", Label: "no", Join: true},
		{Split: "dev", Stratum: "random", Label: "no"},
		{Split: "dev", Stratum: "random", Label: "yes"},
		{Split: "dev", Stratum: "random", Label: "unjudgeable", Join: true},
		{Split: "dev", Stratum: "random", Label: "no", Error: "timeout"},
		{Split: "holdout", Stratum: "spam", Label: "no"},
	} {
		key := result.Split + "/" + result.Stratum
		if stats[key] == nil {
			stats[key] = &gateReplayStats{Split: result.Split, Stratum: result.Stratum}
		}
		stats[key].add(result)
	}
	require.Len(t, stats, 2)
	random := stats["dev/random"].summary()
	assert.Equal(t, 6, random.Cases)
	assert.Equal(t, 1, random.Errors)
	assert.Equal(t, 1, random.Unjudgeable)
	assert.Equal(t, 1, random.Nuisance)
	assert.InDelta(t, 0.5, *random.Precision, 0.0001)
	assert.InDelta(t, 0.5, *random.ReplyRate, 0.0001)
	spam := stats["holdout/spam"].summary()
	assert.Nil(t, spam.Precision)
	assert.Zero(t, *spam.ReplyRate)
	empty := (&gateReplayStats{}).summary()
	assert.Nil(t, empty.Precision)
	assert.Nil(t, empty.ReplyRate)
}

type gateReplayCase struct {
	Version         int           `json:"version"`
	CaseID          string        `json:"case_id"`
	ConversationID  string        `json:"conversation_id"`
	ChatID          int64         `json:"chat_id"`
	At              time.Time     `json:"at"`
	HistorySize     int           `json:"history_size"`
	Current         bot.Message   `json:"current"`
	Context         []bot.Message `json:"context"`
	SourceMissing   []string      `json:"source_missing,omitempty"`
	SourceTruncated bool          `json:"source_truncated,omitempty"`
	Stratum         string        `json:"stratum"`
	Split           string        `json:"split"`
	Label           string        `json:"label"`
	Reason          string        `json:"reason"`
	EvidenceNeeded  []string      `json:"evidence_needed"`
}

type gateReplayIndex struct {
	cases   map[string]bool
	current map[string]bool
	splits  map[string]string
}

func loadGateReplayCases(input io.Reader) ([]gateReplayCase, error) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024+1)
	index := gateReplayIndex{cases: map[string]bool{}, current: map[string]bool{}, splits: map[string]string{}}
	var cases []gateReplayCase
	line := 0
	for scanner.Scan() {
		line++
		if len(scanner.Bytes()) > 1024*1024 {
			return nil, fmt.Errorf("case line %d exceeds 1 MiB", line)
		}
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		decoder := json.NewDecoder(bytes.NewReader(scanner.Bytes()))
		decoder.DisallowUnknownFields()
		var item gateReplayCase
		if err := decoder.Decode(&item); err != nil {
			return nil, fmt.Errorf("case line %d: %w", line, err)
		}
		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			return nil, fmt.Errorf("case line %d must contain one object", line)
		}
		if err := item.validate(); err != nil {
			return nil, fmt.Errorf("case line %d: %w", line, err)
		}
		if err := index.add(item); err != nil {
			return nil, fmt.Errorf("case line %d: %w", line, err)
		}
		cases = append(cases, item)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read cases: %w", err)
	}
	if len(cases) == 0 {
		return nil, fmt.Errorf("case file is empty")
	}
	return cases, nil
}

func (c gateReplayCase) validate() error {
	if c.Version != 1 || strings.TrimSpace(c.CaseID) == "" || strings.TrimSpace(c.ConversationID) == "" {
		return fmt.Errorf("version 1 and nonempty case/conversation IDs are required")
	}
	if c.ChatID >= 0 || c.Current.ChatID != c.ChatID || c.Current.ID <= 0 || c.At.IsZero() || !c.At.Equal(c.Current.Sent) {
		return fmt.Errorf("current message must have a positive ID, matching group chat and decision timestamp")
	}
	if c.HistorySize < 1 || c.HistorySize > 1000 || c.Context == nil || len(c.Context) > c.HistorySize {
		return fmt.Errorf("history_size must be 1..1000 and context must be an array no longer than history_size")
	}
	if direct, _ := (&OpenAI{}).request(c.Current.Text); direct || len(c.Current.Text) < 8 || c.Current.Text == "idle" {
		return fmt.Errorf("current message is not an automatic-reply candidate")
	}
	if c.Stratum != "random" && c.Stratum != "past_reply" && c.Stratum != "spam" {
		return fmt.Errorf("invalid stratum")
	}
	if c.Split != "dev" && c.Split != "holdout" {
		return fmt.Errorf("invalid split")
	}
	if c.Label != "yes" && c.Label != "no" && c.Label != "unjudgeable" {
		return fmt.Errorf("invalid label")
	}
	if strings.TrimSpace(c.Reason) == "" || len(c.EvidenceNeeded) == 0 {
		return fmt.Errorf("reason and evidence_needed are required")
	}
	for _, evidence := range c.EvidenceNeeded {
		switch evidence {
		case "none":
			if len(c.EvidenceNeeded) != 1 {
				return fmt.Errorf("none cannot be combined with other evidence")
			}
		case "history", "media", "private", "live", "unknown":
		default:
			return fmt.Errorf("invalid evidence_needed value")
		}
	}
	if (len(c.SourceMissing) > 0 || c.SourceTruncated) && c.Label != "unjudgeable" {
		return fmt.Errorf("incomplete source requires an unjudgeable label")
	}
	for _, missing := range c.SourceMissing {
		if strings.TrimSpace(missing) == "" {
			return fmt.Errorf("source_missing entries must be nonempty")
		}
	}
	if c.Current.ReplyTo.Sent.After(c.Current.Sent) {
		return fmt.Errorf("current reply parent is from the future")
	}
	return c.validateContext()
}

func (c gateReplayCase) validateContext() error {
	seen := map[string]bool{}
	var previous time.Time
	for _, msg := range c.Context {
		if msg.ID <= 0 || msg.ChatID >= 0 || msg.Sent.IsZero() || msg.Sent.After(c.At) || msg.Sent.Before(previous) {
			return fmt.Errorf("context must contain chronological preceding group messages")
		}
		if msg.ChatID == c.ChatID && msg.ID >= c.Current.ID {
			return fmt.Errorf("context includes current or later message")
		}
		if msg.ReplyTo.Sent.After(msg.Sent) {
			return fmt.Errorf("context reply parent is from the future")
		}
		key := fmt.Sprintf("%d/%d", msg.ChatID, msg.ID)
		if seen[key] {
			return fmt.Errorf("duplicate context message")
		}
		seen[key] = true
		previous = msg.Sent
	}
	return nil
}

func (c gateReplayCase) buffer() LimitedMessageHistory {
	history := NewLimitedMessageHistory(c.HistorySize)
	for _, msg := range c.Context {
		history.Add(msg)
	}
	history.Add(c.Current)
	return history
}

func (i *gateReplayIndex) add(c gateReplayCase) error {
	key := fmt.Sprintf("%d/%d", c.ChatID, c.Current.ID)
	if i.cases[c.CaseID] {
		return fmt.Errorf("duplicate case ID %q", c.CaseID)
	}
	if i.current[key] {
		return fmt.Errorf("duplicate current message")
	}
	i.cases[c.CaseID], i.current[key] = true, true
	keys := []string{"conversation/" + c.ConversationID, fmt.Sprintf("day/%d/%s", c.ChatID, c.At.UTC().Format("2006-01-02"))}
	history := c.buffer()
	for _, msg := range history.messages {
		if msg.ChatID == c.ChatID && c.At.Sub(msg.Sent) <= historyMaxAge {
			keys = append(keys, fmt.Sprintf("message/%d/%d", msg.ChatID, msg.ID))
		}
	}
	for _, key := range keys {
		if split, ok := i.splits[key]; ok && split != c.Split {
			return fmt.Errorf("conversation, day or recent context crosses splits")
		}
		i.splits[key] = c.Split
	}
	return nil
}

type gateReplayView struct {
	CaseID          string   `json:"case_id"`
	State           jevState `json:"state"`
	SourceMissing   []string `json:"source_missing,omitempty"`
	SourceTruncated bool     `json:"source_truncated,omitempty"`
}

func (c gateReplayCase) blindView() gateReplayView {
	history := c.buffer()
	return gateReplayView{CaseID: c.CaseID, State: history.snapshot(c.Current, c.At).jevState(), SourceMissing: c.SourceMissing, SourceTruncated: c.SourceTruncated}
}

type gateReplay struct{ client jevClient }

type gateReplayClient struct {
	next     jevClient
	response jev.Response
	err      error
	called   bool
}

func (c *gateReplayClient) Ask(ctx context.Context, state any, questions map[string]jev.Question) (jev.Response, error) {
	c.called = true
	c.response, c.err = c.next.Ask(ctx, state, questions)
	return c.response, c.err
}

type gateReplayResult struct {
	CaseID     string   `json:"case_id"`
	Split      string   `json:"split"`
	Stratum    string   `json:"stratum"`
	Label      string   `json:"label"`
	Join       bool     `json:"join"`
	Model      string   `json:"model,omitempty"`
	Invites    *float64 `json:"invites,omitempty"`
	Answerable *float64 `json:"answerable,omitempty"`
	Spam       *float64 `json:"spam,omitempty"`
	Error      string   `json:"error,omitempty"`
	Skipped    string   `json:"skipped,omitempty"`
}

func (r gateReplay) run(c gateReplayCase) (gateReplayResult, error) {
	if err := c.validate(); err != nil {
		return gateReplayResult{}, err
	}
	if r.client == nil {
		return gateReplayResult{}, fmt.Errorf("replay requires a Jev client")
	}
	client := &gateReplayClient{next: r.client}
	o := &OpenAI{params: Params{Jev: client}}
	history := c.buffer()
	snapshot := history.snapshot(c.Current, c.At)
	result := gateReplayResult{CaseID: c.CaseID, Split: c.Split, Stratum: c.Stratum, Label: c.Label}
	result.Join = o.shouldJoin(snapshot)
	if !client.called {
		result.Skipped = "oversized"
		return result, nil
	}
	if client.err != nil {
		result.Error = client.err.Error()
		return result, nil
	}
	if _, err := o.gateScores(client.response); err != nil {
		result.Error = err.Error()
		return result, nil
	}
	result.Model = client.response.Model
	result.Invites = client.response.Answers["invites"].Noul
	result.Answerable = client.response.Answers["answerable"].Noul
	result.Spam = client.response.Answers["spam"].Noul
	return result, nil
}

type gateReplayStats struct {
	Split               string `json:"split"`
	Stratum             string `json:"stratum"`
	Cases               int    `json:"cases"`
	Errors              int    `json:"errors"`
	Unjudgeable         int    `json:"unjudgeable"`
	AcceptedUnjudgeable int    `json:"accepted_unjudgeable"`
	TP                  int    `json:"tp"`
	FP                  int    `json:"fp"`
	TN                  int    `json:"tn"`
	FN                  int    `json:"fn"`
}

func (s *gateReplayStats) add(r gateReplayResult) {
	s.Cases++
	if r.Label == "unjudgeable" {
		s.Unjudgeable++
	}
	if r.Error != "" {
		s.Errors++
		return
	}
	if r.Label == "unjudgeable" {
		if r.Join {
			s.AcceptedUnjudgeable++
		}
		return
	}
	switch {
	case r.Join && r.Label == "yes":
		s.TP++
	case r.Join:
		s.FP++
	case r.Label == "yes":
		s.FN++
	default:
		s.TN++
	}
}

type gateReplaySummary struct {
	gateReplayStats
	Precision *float64 `json:"precision"`
	ReplyRate *float64 `json:"reply_rate"`
	Nuisance  int      `json:"nuisance"`
}

func (s *gateReplayStats) summary() gateReplaySummary {
	result := gateReplaySummary{gateReplayStats: *s, Nuisance: s.FP}
	if accepted := s.TP + s.FP; accepted > 0 {
		precision := float64(s.TP) / float64(accepted)
		result.Precision = &precision
	}
	if scored := s.TP + s.FP + s.TN + s.FN; scored > 0 {
		rate := float64(s.TP+s.FP) / float64(scored)
		result.ReplyRate = &rate
	}
	return result
}

func TestGateReplayViews(t *testing.T) {
	path := os.Getenv("JEV_REPLAY_CASES")
	if path == "" {
		t.Skip("set JEV_REPLAY_CASES to render blinded views without API calls")
	}
	file, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	cases, err := loadGateReplayCases(file)
	require.NoError(t, err)
	for _, item := range cases {
		data, err := json.Marshal(item.blindView())
		require.NoError(t, err)
		t.Logf("view %s", data)
	}
}

func TestJevLiveReplay(t *testing.T) {
	path := os.Getenv("JEV_LIVE_CASES")
	if path == "" {
		t.Skip("set JEV_LIVE_CASES to opt into live Jev replay")
	}
	file, err := os.Open(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = file.Close() })
	cases, err := loadGateReplayCases(file)
	require.NoError(t, err)
	split := os.Getenv("JEV_LIVE_SPLIT")
	if split == "" {
		split = "dev"
	}
	require.Contains(t, []string{"dev", "holdout"}, split)
	key := os.Getenv("JEV_KEY")
	require.NotEmpty(t, key, "JEV_KEY must be supplied privately")
	model := os.Getenv("JEV_MODEL")
	if model == "" {
		model = "jev-1.13.0"
	}
	timeout := 2 * time.Second
	if value := os.Getenv("JEV_TIMEOUT"); value != "" {
		timeout, err = time.ParseDuration(value)
		require.NoError(t, err)
	}
	require.Positive(t, timeout)
	replay := gateReplay{client: jev.New(jev.Params{Key: key, Model: model, Timeout: timeout})}
	stats := map[string]*gateReplayStats{}
	t.Logf("config split=%s requested_model=%s invites=%g answerable=%g spam=%g", split, model, invitesThreshold, answerableThreshold, spamThreshold)
	for _, item := range cases {
		if item.Split != split {
			continue
		}
		result, err := replay.run(item)
		require.NoError(t, err)
		data, err := json.Marshal(result)
		require.NoError(t, err)
		t.Logf("case %s", data)
		group := item.Split + "/" + item.Stratum
		if stats[group] == nil {
			stats[group] = &gateReplayStats{Split: item.Split, Stratum: item.Stratum}
		}
		stats[group].add(result)
		if result.Error != "" {
			t.Errorf("case %q could not be scored", item.CaseID)
		}
	}
	require.NotEmpty(t, stats, "no cases in selected split")
	groups := make([]string, 0, len(stats))
	for group := range stats {
		groups = append(groups, group)
	}
	sort.Strings(groups)
	for _, group := range groups {
		data, err := json.Marshal(stats[group].summary())
		require.NoError(t, err)
		t.Logf("summary %s", data)
	}
}
