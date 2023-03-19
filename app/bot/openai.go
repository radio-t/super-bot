package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

//go:generate moq --out mocks/openai_client.go --pkg mocks --skip-ensure . OpenAIClient:OpenAIClient

// OpenAIClient is interface for OpenAI client with the possibility to mock it
type OpenAIClient interface {
	CreateChatCompletion(context.Context, openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

// OpenAI bot, returns responses from ChatGPT via OpenAI API
type OpenAI struct {
	authToken string
	client    OpenAIClient
	maxTokens int
	prompt    string

	nowFn  func() time.Time // for testing
	lastDT time.Time
}

// NewOpenAI makes a bot for ChatGPT
// maxTokens is hard limit for the number of tokens in the response
// https://platform.openai.com/docs/api-reference/chat/create#chat/create-max_tokens
func NewOpenAI(authToken string, maxTokens int, prompt string, httpClient *http.Client) *OpenAI {
	log.Printf("[INFO] OpenAI bot with github.com/sashabaranov/go-openai, prompt=%s, max=%d", prompt, maxTokens)
	config := openai.DefaultConfig(authToken)
	config.HTTPClient = httpClient

	client := openai.NewClientWithConfig(config)
	return &OpenAI{authToken: authToken, client: client, maxTokens: maxTokens, prompt: prompt, nowFn: time.Now}
}

// OnMessage pass msg to all bots and collects responses
func (o *OpenAI) OnMessage(msg Message) (response Response) {
	ok, reqText := o.request(msg.Text)
	if !ok {
		return Response{}
	}

	if o.nowFn().Sub(o.lastDT) < 30*time.Minute {
		log.Printf("[WARN] OpenAI bot is too busy, last request was %s ago, %s banned", time.Since(o.lastDT), msg.From.Username)
		return Response{
			Text: fmt.Sprintf("Слишком много запросов, следующий запрос можно будет сделать через %d минут."+
				"\n%s получает бан на 1 час.", int(30-time.Since(o.lastDT).Minutes()), msg.From.Username),
			Send:        true,
			BanInterval: time.Hour,
			User:        msg.From,
			ReplyTo:     msg.ID, // reply to the message
		}
	}

	responseAI, err := o.chatGPTRequest(reqText)
	if err != nil {
		log.Printf("[WARN] failed to make request to ChatGPT '%s', error=%v", reqText, err)
		return Response{}
	}

	o.lastDT = o.nowFn()
	log.Printf("[DEBUG] next request to ChatGPT can be made after %s, in %d minutes",
		o.lastDT.Add(30*time.Minute), int(30-time.Since(o.lastDT).Minutes()))
	return Response{
		Text:    responseAI,
		Send:    true,
		ReplyTo: msg.ID, // reply to the message
	}
}

func (o *OpenAI) request(text string) (react bool, reqText string) {
	for _, prefix := range o.ReactOn() {
		if strings.HasPrefix(text, prefix) {
			return true, strings.TrimSpace(strings.TrimPrefix(text, prefix))
		}
	}
	return false, ""
}

// Help returns help message
func (o *OpenAI) Help() string {
	return genHelpMsg(o.ReactOn(), "Спросите что-нибудь у ChatGPT")
}

func (o *OpenAI) chatGPTRequest(request string) (response string, err error) {

	r := request
	if o.prompt != "" {
		r = o.prompt + ".\n" + request
	}
	resp, err := o.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:     openai.GPT3Dot5Turbo,
			MaxTokens: o.maxTokens,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You answer with no more than 50 words",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: r,
				},
			},
		},
	)

	if err != nil {
		return "", err
	}

	// OpenAI platform supports to return multiple chat completion choices
	// but we use only the first one
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-n
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}

// ReactOn keys
func (o *OpenAI) ReactOn() []string {
	return []string{"chat!", "gpt!", "ai!", "чат!"}
}
