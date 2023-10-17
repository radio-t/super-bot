package bot

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/sashabaranov/go-openai"
)

//go:generate moq --out mocks/openai_client.go --pkg mocks --skip-ensure . OpenAIClient

// SpamOpenAIFilter bot, checks if user is a spammer using openai api call
type SpamOpenAIFilter struct {
	dry          bool
	superUser    SuperUser
	openaiClient OpenAIClient
	milMsgLen    int

	spamPrompt    string
	enabled       bool
	approvedUsers map[int64]bool
}

// OpenAIClient is interface for OpenAI client with the possibility to mock it
type OpenAIClient interface {
	CreateChatCompletion(context.Context, openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

// NewSpamOpenAIFilter makes a spam detecting bot
func NewSpamOpenAIFilter(spamSamples io.Reader, openaiClient OpenAIClient, maxPromptLen int, superUser SuperUser, minMsgLen int, dry bool) *SpamOpenAIFilter {
	log.Printf("[INFO] Spam bot (openai): max prompt len=%d, min msg: %d, dry=%v", maxPromptLen, minMsgLen, dry)
	res := &SpamOpenAIFilter{dry: dry, approvedUsers: map[int64]bool{}, superUser: superUser, openaiClient: openaiClient}

	scanner := bufio.NewScanner(spamSamples)
	for scanner.Scan() {
		res.spamPrompt += scanner.Text() + "\n"
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[WARN] failed to read spam samples, error=%v", err)
		res.enabled = false
	} else {
		res.enabled = true
	}
	log.Printf("[DEBUG] spam initial prompt: %d", len(res.spamPrompt))
	if len(res.spamPrompt) > maxPromptLen {
		res.spamPrompt = res.spamPrompt[:maxPromptLen]
	}
	log.Printf("[DEBUG] spam prompt len: %d", len(res.spamPrompt))
	return res
}

// OnMessage checks if user already approved and if not checks if user is a spammer
func (s *SpamOpenAIFilter) OnMessage(msg Message) (response Response) {
	if s.approvedUsers[msg.From.ID] || msg.From.ID == 0 || len(msg.Text) < s.milMsgLen || !s.enabled {
		return Response{}
	}

	if s.superUser.IsSuper(msg.From.Username) {
		return Response{} // don't check super users for spam
	}
	displayUsername := DisplayName(msg)
	if !s.isSpam(msg.Text) {
		log.Printf("[INFO] user %s (%d) is not a spammer, added to aproved", displayUsername, msg.From.ID)
		s.approvedUsers[msg.From.ID] = true
		return Response{} // not a spam
	}

	log.Printf("[INFO] user %q detected as spammer by openai, msg: %q", displayUsername, msg.Text)
	if s.dry {
		return Response{
			Text: fmt.Sprintf("this is spam (openai) from %q, but I'm in dry mode, so I'll do nothing yet", displayUsername),
			Send: true, ReplyTo: msg.ID,
		}
	}
	return Response{Text: "this is spam (openai)! go to ban, " + displayUsername, Send: true, ReplyTo: msg.ID,
		BanInterval: permanentBanDuration, DeleteReplyTo: true}
}

// Help returns help message
func (s *SpamOpenAIFilter) Help() string { return "" }

// ReactOn keys
func (s *SpamOpenAIFilter) ReactOn() []string { return []string{} }

// isSpam checks if a given message is similar to any of the known bad messages.
func (s *SpamOpenAIFilter) isSpam(message string) bool {

	messages := []openai.ChatCompletionMessage{}
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "This is the list of spam messages. I will give you a message to detect if it is spam or not and you will answer with a single world \"OK\" or \"SPAM\". Answer \"SPAM\" only if you absolutely certain it is spam\n\n" + s.spamPrompt,
	}, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message,
	})

	resp, err := s.openaiClient.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:     openai.GPT4,
			MaxTokens: 100,
			Messages:  messages,
		},
	)
	if err != nil {
		log.Printf("[WARN] failed to check spam, error=%v", err)
		return false
	}
	if len(resp.Choices) == 0 {
		log.Printf("[WARN] empty response from openai")
		return false
	}

	log.Printf("[DEBUG] openai response: %q", resp.Choices[0].Message.Content)
	return strings.Contains(resp.Choices[0].Message.Content, "SPAM")
}