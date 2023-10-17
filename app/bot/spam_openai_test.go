package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestSpamOpenAIFilter_isSpam(t *testing.T) {
	spamSamples := strings.NewReader("spam1\nspam2\nspam3")
	mockOpenAI := &mocks.OpenAIClientMock{}

	filter := NewSpamOpenAIFilter(spamSamples, mockOpenAI, 4096, nil, 5, false)
	require.True(t, filter.enabled)
	assert.True(t, len(filter.spamPrompt) <= 4096)

	mockOpenAI.CreateChatCompletionFunc = func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
		return openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "SPAM",
					},
				},
			},
		}, nil
	}

	assert.True(t, filter.isSpam("this is a spam message"))
	assert.Equal(t, 1, len(mockOpenAI.CreateChatCompletionCalls()))

	mockOpenAI.CreateChatCompletionFunc = func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
		return openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{
					Message: openai.ChatCompletionMessage{
						Content: "OK",
					},
				},
			},
		}, nil
	}

	assert.False(t, filter.isSpam("this is not a spam message"))
	assert.Equal(t, 2, len(mockOpenAI.CreateChatCompletionCalls()))
}

func TestSpamOpenAIFilter_OnMessage(t *testing.T) {
	superUser := &mocks.SuperUser{IsSuperFunc: func(userName string) bool {
		if userName == "super" || userName == "admin" {
			return true
		}
		return false
	}}

	spamSamples := strings.NewReader("spam1\nspam2\nspam3")
	mockOpenAI := &mocks.OpenAIClientMock{
		CreateChatCompletionFunc: func(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
			return openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: "OK",
						},
					},
				},
			}, nil
		},
	}

	filter := NewSpamOpenAIFilter(spamSamples, mockOpenAI, 4096, superUser, 5, false)

	msg := Message{From: User{ID: 1, Username: "user1"}, Text: "hello"}
	resp := filter.OnMessage(msg)

	assert.Empty(t, resp.Text)
}
