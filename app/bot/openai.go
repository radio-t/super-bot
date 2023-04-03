package bot

import (
	"context"
	"fmt"
	"log"
	"math/rand"
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

// OpenAIParams contains parameters for OpenAI bot
type OpenAIParams struct {
	AuthToken               string
	MaxTokens               int
	Prompt                  string
	EnableAutoResponse      bool
	HistorySize             int
	HistoryReplyProbability int // Percentage of the probability to reply with history
}

// OpenAI bot, returns responses from ChatGPT via OpenAI API
type OpenAI struct {
	client OpenAIClient

	params    OpenAIParams
	superUser SuperUser

	history LimitedMessageHistory
	rand    func(n int64) int64 // tests may change it

	nowFn  func() time.Time // for testing
	lastDT time.Time
}

var maxMsgLen = 14000

// NewOpenAI makes a bot for ChatGPT
// MaxTokens is hard limit for the number of tokens in the response
// https://platform.openai.com/docs/api-reference/chat/create#chat/create-max_tokens
func NewOpenAI(params OpenAIParams, httpClient *http.Client, superUser SuperUser) *OpenAI {
	log.Printf("[INFO] OpenAI bot with github.com/sashabaranov/go-openai, Prompt=%s, max=%d. Auto response is %v",
		params.Prompt, params.MaxTokens, params.EnableAutoResponse)

	openAIConfig := openai.DefaultConfig(params.AuthToken)
	openAIConfig.HTTPClient = httpClient
	client := openai.NewClientWithConfig(openAIConfig)
	history := NewLimitedMessageHistory(params.HistorySize)

	return &OpenAI{client: client, params: params, superUser: superUser,
		history: history, rand: rand.Int63n, nowFn: time.Now}
}

// OnMessage pass msg to all bots and collects responses
func (o *OpenAI) OnMessage(msg Message) (response Response) {
	ok, reqText := o.request(msg.Text)
	if !ok {
		if !o.params.EnableAutoResponse || msg.Text == "idle" || len(msg.Text) < 8 {
			// don't answer on short messages or "idle" command or if auto response is disabled
			return Response{}
		}

		// All the non-matching requests processed for the reactions based on the history.
		// save message to history and answer with ChatGPT if needed
		o.history.Add(msg)

		if !o.shouldAnswerWithHistory(msg) {
			return Response{}
		}

		responseAI, err := o.chatGPTRequestWithHistory("You answer with no more than 50 words, should be in Russian language")
		if err != nil {
			log.Printf("[WARN] failed to make context request to ChatGPT error=%v", err)
			return Response{}
		}
		log.Printf("[DEBUG] OpenAI bot answer with history: %q", responseAI)
		return Response{
			Text: responseAI,
			Send: true,
		}
	}

	if ok, banMessage := o.checkRequest(msg.From.Username, reqText); !ok {
		return Response{
			Text:        banMessage,
			Send:        true,
			BanInterval: time.Hour,
			User:        msg.From,
			ReplyTo:     msg.ID, // reply to the message
		}
	}

	responseAI, err := o.chatGPTRequest(reqText, o.params.Prompt, "You answer with no more than 50 words")
	if err != nil {
		log.Printf("[WARN] failed to make request to ChatGPT '%s', error=%v", reqText, err)
		return Response{}
	}

	if ok, banMessage := o.checkResponseAI(msg.From.Username, responseAI); !ok {
		return Response{
			Text:        banMessage,
			Send:        true,
			BanInterval: time.Hour,
			User:        msg.From,
			ReplyTo:     msg.ID, // reply to the message
		}
	}

	if !o.superUser.IsSuper(msg.From.Username) {
		o.lastDT = o.nowFn() // don't update lastDT for super users
	}

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

func (o *OpenAI) checkRequest(username, text string) (ok bool, banMessage string) {
	if o.superUser.IsSuper(username) {
		return true, ""
	}

	wtfContains := WTFSteroidChecker{message: text}

	if wtfContains.ContainsWTF() {
		log.Printf("[WARN] OpenAI bot has wtf request, %s banned", username)
		reason := "Вы знаете правила"
		return false, fmt.Sprintf("%s\n@%s получает бан на 1 час.", reason, username)
	}

	if o.nowFn().Sub(o.lastDT) < 30*time.Minute {
		log.Printf("[WARN] OpenAI bot is too busy, last request was %s ago, %s banned", time.Since(o.lastDT), username)
		reason := fmt.Sprintf("Слишком много запросов, следующий запрос можно будет сделать через %d минут.",
			int(30-time.Since(o.lastDT).Minutes()))

		return false, fmt.Sprintf("%s\n@%s получает бан на 1 час.", reason, username)
	}

	return true, ""
}

func (o *OpenAI) checkResponseAI(username, responseAI string) (ok bool, banMessage string) {
	if o.superUser.IsSuper(username) {
		return true, ""
	}

	wtfContains := WTFSteroidChecker{message: responseAI}

	if wtfContains.ContainsWTF() {
		log.Printf("[WARN] OpenAI bot response contains wtf, User %s banned", username)
		return false, fmt.Sprintf("@%s выиграл в лотерею и получает бан на 1 час.", username)
	}

	return true, ""
}

// Help returns help message
func (o *OpenAI) Help() string {
	return genHelpMsg(o.ReactOn(), "Спросите что-нибудь у ChatGPT")
}

func (o *OpenAI) chatGPTRequest(request, userPrompt, sysPrompt string) (response string, err error) {
	r := request
	if userPrompt != "" {
		r = userPrompt + ".\n" + request
	}

	if len(r) > maxMsgLen {
		r = r[:maxMsgLen]
	}

	return o.chatGPTRequestInternal([]openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: sysPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: r,
		},
	})
}

func (o *OpenAI) shouldAnswerWithHistory(msg Message) bool {
	if o.history.count < o.history.limit {
		return false
	}

	if len(msg.Text) > 0 && msg.Text[len(msg.Text)-1:] != "?" { // don't try to answer to short messages, like wtf?
		return false
	}

	// by default 10% chance to answer with ChatGPT for question
	return o.rand(100) < int64(o.params.HistoryReplyProbability)
}

func (o *OpenAI) chatGPTRequestWithHistory(sysPrompt string) (response string, err error) {
	messages := make([]openai.ChatCompletionMessage, 0, len(o.history.messages)+1)

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: sysPrompt,
	})

	for _, message := range o.history.messages {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: message.Text,
		})
	}

	return o.chatGPTRequestInternal(messages)
}

func (o *OpenAI) chatGPTRequestInternal(messages []openai.ChatCompletionMessage) (response string, err error) {

	resp, err := o.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:     openai.GPT3Dot5Turbo,
			MaxTokens: o.params.MaxTokens,
			Messages:  messages,
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

// Summary returns summary of the text
func (o *OpenAI) Summary(text string) (response string, err error) {
	return o.chatGPTRequest(text, "", "Make a short summary, up to 50 words, followed by a list of bullet points. Each bullet point is limited to 50 words, up to 7 in total. All in markdown format and translated to russian:\n")
}

// ReactOn keys
func (o *OpenAI) ReactOn() []string {
	return []string{"chat!", "gpt!", "ai!", "чат!"}
}
