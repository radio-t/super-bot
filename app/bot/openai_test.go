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
		{"Good result", "prompt", jsonResponse, true, Response{Text: "Mock response", Send: true, ReplyTo: 756}},
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

			o := NewOpenAI("ss-mockToken", 100, tt.prompt, &http.Client{Timeout: 10 * time.Second}, su)
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

	o := NewOpenAI("ss-mockToken", 100, "", &http.Client{Timeout: 10 * time.Second}, su)
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
