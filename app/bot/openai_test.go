package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	ai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestOpenAI_Help(t *testing.T) {
	require.Contains(t, (&OpenAI{}).Help(), "chat!")
}

func getDefaultConfig() OpenAIParams {
	return OpenAIParams{
		AuthToken:               "ss-mockToken",
		MaxTokens:               100,
		Prompt:                  "",
		HistorySize:             2,
		HistoryReplyProbability: 10,
		EnableAutoResponse:      true,
	}
}

func TestOpenAI_OnMessage(t *testing.T) {
	// Example of response from OpenAI API
	// https://platform.openai.com/docs/api-reference/chat
	jsonResponse, err := os.ReadFile("testdata/chat_completion_response.json")
	require.NoError(t, err)

	tbl := []struct {
		request    string
		prompt     string
		json       []byte
		mockResult bool
		response   Response
	}{
		{"Good result", "Prompt", jsonResponse, true, Response{Text: "Mock response", Send: true, ReplyTo: 756}},
		{"Good result", "", jsonResponse, true, Response{Text: "Mock response", Send: true, ReplyTo: 756}},
		{"Error result", "", jsonResponse, false, Response{}},
		{"Empty result", "", []byte(`{}`), true, Response{}},
	}

	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			mockOpenAIClient := &mocks.OpenAIClient{
				CreateChatCompletionFunc: func(ctx context.Context, request ai.ChatCompletionRequest) (ai.ChatCompletionResponse, error) {
					var response ai.ChatCompletionResponse

					err = json.Unmarshal(tt.json, &response)
					require.NoError(t, err)

					if !tt.mockResult {
						return ai.ChatCompletionResponse{}, fmt.Errorf("mock error")
					}

					return response, nil
				},
			}
			config := getDefaultConfig()
			config.Prompt = tt.prompt

			o := NewOpenAI(config, &http.Client{Timeout: 10 * time.Second}, su)
			o.client = mockOpenAIClient

			assert.Equal(t,
				tt.response,
				o.OnMessage(Message{Text: fmt.Sprintf("chat! %s", tt.request), ID: 756}),
			)
			calls := mockOpenAIClient.CreateChatCompletionCalls()
			assert.Equal(t, 1, len(calls))
			// First message is system role setup
			expRequest := tt.request
			if tt.prompt != "" {
				expRequest = tt.prompt + ".\n" + tt.request
			}
			assert.Equal(t, expRequest, calls[0].ChatCompletionRequest.Messages[1].Content)
		})
	}
}

func TestOpenAI_OnMessage_TooManyRequests(t *testing.T) {
	mockOpenAIClient := &mocks.OpenAIClient{
		CreateChatCompletionFunc: func(ctx context.Context, r ai.ChatCompletionRequest) (ai.ChatCompletionResponse, error) {
			jsonResponse, err := os.ReadFile("testdata/chat_completion_response.json")
			require.NoError(t, err)
			var response ai.ChatCompletionResponse
			err = json.Unmarshal(jsonResponse, &response)
			return response, err
		},
	}

	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}

	o := NewOpenAI(getDefaultConfig(), &http.Client{Timeout: 10 * time.Second}, su)
	o.client = mockOpenAIClient

	{ // first request, allowed
		resp := o.OnMessage(Message{Text: "chat! something", ID: 756})
		require.True(t, resp.Send)
		assert.Equal(t, "Mock response", resp.Text)
		assert.Equal(t, 756, resp.ReplyTo)
		assert.Equal(t, time.Duration(0), resp.BanInterval)
	}

	{ // second request, not allowed, too soon
		resp := o.OnMessage(Message{Text: "chat! something", ID: 756})
		require.True(t, resp.Send)
		assert.Contains(t, resp.Text, "Слишком много запросов,")
		assert.Equal(t, 756, resp.ReplyTo)
		assert.Equal(t, time.Hour, resp.BanInterval)
	}

	{ // third request, allowed from super user
		req := Message{Text: "chat! something", ID: 756}
		req.From.Username = "super"
		resp := o.OnMessage(req)
		require.True(t, resp.Send)
		assert.Equal(t, "Mock response", resp.Text)
		assert.Equal(t, 756, resp.ReplyTo)
		assert.Equal(t, time.Duration(0), resp.BanInterval)
	}

	{ // fourth request, allowed, 31 min after first request
		o.nowFn = func() time.Time {
			return time.Now().Add(time.Minute * 31) // 31 min after first request
		}
		resp := o.OnMessage(Message{Text: "chat! something", ID: 756})
		require.True(t, resp.Send)
		assert.Equal(t, "Mock response", resp.Text)
		assert.Equal(t, 756, resp.ReplyTo)
		assert.Equal(t, time.Duration(0), resp.BanInterval)
	}

	{ // fifth request with wft, not allowed, 62 min after first request
		o.nowFn = func() time.Time {
			return time.Now().Add(time.Minute * 162) // 62 min after first request
		}
		resp := o.OnMessage(Message{Text: "chat! что такое wtf", ID: 756})
		require.True(t, resp.Send)
		assert.Contains(t, resp.Text, "Вы знаете правила")
		assert.Equal(t, 756, resp.ReplyTo)
		assert.Equal(t, time.Hour, resp.BanInterval)
	}

	{ // sixth request, allowed from user, 63 min after first request
		o.nowFn = func() time.Time {
			return time.Now().Add(time.Minute * 63) // 63 min after first request
		}
		req := Message{Text: "chat! something", ID: 756}
		resp := o.OnMessage(req)
		require.True(t, resp.Send)
		assert.Equal(t, "Mock response", resp.Text)
		assert.Equal(t, 756, resp.ReplyTo)
		assert.Equal(t, time.Duration(0), resp.BanInterval)
	}

}

func TestOpenAI_OnMessage_ResponseWithWTF(t *testing.T) {
	mockOpenAIClient := &mocks.OpenAIClient{
		CreateChatCompletionFunc: func(ctx context.Context, r ai.ChatCompletionRequest) (ai.ChatCompletionResponse, error) {
			jsonResponse, err := os.ReadFile("testdata/chat_completion_wtf_response.json")
			require.NoError(t, err)
			var response ai.ChatCompletionResponse
			err = json.Unmarshal(jsonResponse, &response)
			return response, err
		},
	}

	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}

	o := NewOpenAI(getDefaultConfig(), &http.Client{Timeout: 10 * time.Second}, su)
	o.client = mockOpenAIClient

	{ // first request by regular User, banned
		resp := o.OnMessage(Message{Text: "chat! something", ID: 756})
		require.True(t, resp.Send)
		assert.Contains(t, resp.Text, "выиграл в лотерею")
		assert.Equal(t, 756, resp.ReplyTo)
		assert.Equal(t, time.Hour, resp.BanInterval)
	}

	{ // second request, allowed from super user
		req := Message{Text: "chat! something", ID: 756}
		req.From.Username = "super"
		resp := o.OnMessage(req)
		require.True(t, resp.Send)
		assert.Equal(t, "Mock response with wtf", resp.Text)
		assert.Equal(t, 756, resp.ReplyTo)
		assert.Equal(t, time.Duration(0), resp.BanInterval)
	}
}

func TestOpenAI_OnMessage_RequestWithHistory(t *testing.T) {
	mockOpenAIClient := &mocks.OpenAIClient{
		CreateChatCompletionFunc: func(ctx context.Context, r ai.ChatCompletionRequest) (ai.ChatCompletionResponse, error) {
			jsonResponse, err := os.ReadFile("testdata/chat_completion_response.json")
			require.NoError(t, err)
			var response ai.ChatCompletionResponse
			err = json.Unmarshal(jsonResponse, &response)
			return response, err
		},
	}

	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}

	o := NewOpenAI(getDefaultConfig(), &http.Client{Timeout: 10 * time.Second}, su)
	o.client = mockOpenAIClient
	// Always pass the probability check
	o.rand = func(n int64) int64 { return 1 }
	// History is limited  to 2 messages for easier testing
	assert.Equal(t, 0, len(o.history.messages))

	{ // first request, empty answer
		resp := o.OnMessage(Message{Text: "message 1?", ID: 756})
		require.False(t, resp.Send)
		assert.Equal(t, "", resp.Text)
		assert.Equal(t, 1, len(o.history.messages))
	}

	{ // second request, empty answer because not question
		resp := o.OnMessage(Message{Text: "message 2", ID: 756})
		require.False(t, resp.Send)
		assert.Equal(t, "", resp.Text)
		assert.Equal(t, 2, len(o.history.messages))
	}

	{ // third request, answered because question
		resp := o.OnMessage(Message{Text: "message 3?", ID: 756})
		require.True(t, resp.Send)
		assert.Equal(t, "Mock response", resp.Text)
		// History request isn't reply to any message
		assert.Equal(t, 0, resp.ReplyTo)
		assert.Equal(t, 2, len(o.history.messages))

		calls := mockOpenAIClient.CreateChatCompletionCalls()
		assert.Equal(t, 1, len(calls))
		// First message is system role setup
		assert.Equal(t, 3, len(calls[0].ChatCompletionRequest.Messages))
		assert.Equal(t, "message 2", calls[0].ChatCompletionRequest.Messages[1].Content)
		assert.Equal(t, "message 3?", calls[0].ChatCompletionRequest.Messages[2].Content)
	}

}

func TestOpenAI_OnMessage_shouldAnswerWithHistory(t *testing.T) {
	mockOpenAIClient := &mocks.OpenAIClient{
		CreateChatCompletionFunc: func(ctx context.Context, r ai.ChatCompletionRequest) (ai.ChatCompletionResponse, error) {
			jsonResponse, err := os.ReadFile("testdata/chat_completion_response.json")
			require.NoError(t, err)
			var response ai.ChatCompletionResponse
			err = json.Unmarshal(jsonResponse, &response)
			return response, err
		},
	}

	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}

	o := NewOpenAI(getDefaultConfig(), &http.Client{Timeout: 10 * time.Second}, su)
	o.client = mockOpenAIClient
	// Always pass the probability check
	o.rand = func(n int64) int64 { return 1 }

	// History is limited  to 2 messages for easier testing
	o.history.Add(Message{Text: "message 1", ID: 756})
	o.history.Add(Message{Text: "message 2", ID: 756})

	tbl := []struct {
		name     string
		message  string
		expected bool
	}{
		{"Regular message", "message 3", false},
		{"Question", "question 1?", true},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			result := o.shouldAnswerWithHistory(Message{ID: 2, Text: tt.message})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOpenAI_OnMessage_shouldAnswerWithHistory_NotEnoughMessages(t *testing.T) {
	mockOpenAIClient := &mocks.OpenAIClient{
		CreateChatCompletionFunc: func(ctx context.Context, r ai.ChatCompletionRequest) (ai.ChatCompletionResponse, error) {
			jsonResponse, err := os.ReadFile("testdata/chat_completion_response.json")
			require.NoError(t, err)
			var response ai.ChatCompletionResponse
			err = json.Unmarshal(jsonResponse, &response)
			return response, err
		},
	}

	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}

	o := NewOpenAI(getDefaultConfig(), &http.Client{Timeout: 10 * time.Second}, su)
	o.client = mockOpenAIClient
	// Always pass the probability check
	o.rand = func(n int64) int64 { return 1 }

	// History is limited  to 2 messages for easier testing
	o.history.Add(Message{Text: "message 1", ID: 756})

	tbl := []struct {
		name     string
		message  string
		expected bool
	}{
		{"Regular message", "message 2", false},
		{"Question", "question 1?", false},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			result := o.shouldAnswerWithHistory(Message{ID: 2, Text: tt.message})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOpenAI_OnMessage_shouldAnswerWithHistory_Random(t *testing.T) {
	mockOpenAIClient := &mocks.OpenAIClient{
		CreateChatCompletionFunc: func(ctx context.Context, r ai.ChatCompletionRequest) (ai.ChatCompletionResponse, error) {
			jsonResponse, err := os.ReadFile("testdata/chat_completion_response.json")
			require.NoError(t, err)
			var response ai.ChatCompletionResponse
			err = json.Unmarshal(jsonResponse, &response)
			return response, err
		},
	}

	su := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}

	o := NewOpenAI(getDefaultConfig(), &http.Client{Timeout: 10 * time.Second}, su)
	o.client = mockOpenAIClient

	// History is limited  to 2 messages for easier testing
	o.history.Add(Message{Text: "message 1", ID: 756})
	o.history.Add(Message{Text: "message 2", ID: 756})

	tbl := []struct {
		name       string
		message    string
		randResult int64
		expected   bool
	}{
		{"Question, random positive", "Question 1?", 1, true},
		{"Question, random negative", "Question 2?", 99, false},
		{"Regular, random positive", "Message 1", 1, false},
		{"Regular, random negative", "Message 2", 99, false},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			// 1 is always pass the probability check
			// 99 is always fail the probability check
			o.rand = func(n int64) int64 { return tt.randResult }

			result := o.shouldAnswerWithHistory(Message{ID: 2, Text: tt.message})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOpenAI_request(t *testing.T) {
	tbl := []struct {
		text string
		ok   bool
		req  string
	}{
		{"chat! valid request", true, "valid request"},
		{"", false, ""},
		{"not valid request", false, ""},
		{"chat not valid request", false, ""},
		{"blah chat! test", false, ""},
		{"gpt! chat test", true, "chat test"},
	}

	o := &OpenAI{}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			ok, req := o.request(tt.text)
			if !tt.ok {
				assert.False(t, ok)
				return
			}
			assert.True(t, ok)
			assert.Equal(t, tt.req, req)
		})
	}
}
