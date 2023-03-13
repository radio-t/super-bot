package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/radio-t/super-bot/app/bot/mocks"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		json       []byte
		mockResult bool
		response   Response
	}{
		{"Good result", jsonResponse, true, Response{Text: "Mock response", Send: true}},
		{"Error result", jsonResponse, false, Response{}},
		{"Empty result", []byte(`{}`), true, Response{}},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			mockOpenAIClient := &mocks.OpenAIClient{
				CreateChatCompletionFunc: func(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
					var response openai.ChatCompletionResponse

					err = json.Unmarshal(tt.json, &response)
					require.NoError(t, err)

					if !tt.mockResult {
						return openai.ChatCompletionResponse{}, fmt.Errorf("mock error")
					}

					return response, nil
				},
			}

			o := &OpenAI{
				authToken: "ss-mockToken",
				client:    mockOpenAIClient,
			}

			assert.Equal(t,
				tt.response,
				o.OnMessage(Message{Text: fmt.Sprintf("chat! %s", tt.request)}),
			)
			calls := mockOpenAIClient.CreateChatCompletionCalls()
			assert.Equal(t, 1, len(calls))
			assert.Equal(t, tt.request, calls[0].ChatCompletionRequest.Messages[0].Content)
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
