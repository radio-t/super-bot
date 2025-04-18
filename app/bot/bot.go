package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-pkgz/syncs"
)

//go:generate moq --out mocks/http_client.go --pkg mocks --skip-ensure . HTTPClient:HTTPClient
//go:generate moq --out mock_interface.go . Interface
//go:generate moq --out mocks/super_user.go --pkg mocks --skip-ensure . SuperUser:SuperUser

// GenHelpMsg construct help message from bot's ReactOn
func GenHelpMsg(com []string, msg string) string {
	return EscapeMarkDownV1Text(strings.Join(com, ", ")) + " _– " + msg + "_\n"
}

// Interface is a bot reactive spec. response will be sent if "send" result is true
type Interface interface {
	OnMessage(msg Message) (response Response)
	ReactOn() []string
	Help() string
}

// Response describes bot's answer on particular message
type Response struct {
	Text          string
	Send          bool          // status
	Pin           bool          // enable pin
	Unpin         bool          // enable unpin
	Preview       bool          // enable web preview
	BanInterval   time.Duration // bots banning user set the interval
	User          User          // user to ban
	ChannelID     int64         // channel to ban, if set then User and BanInterval are ignored
	ReplyTo       int           // message to reply to, if 0 then no reply but common message
	ParseMode     string        // parse mode for message in Telegram (we use Markdown by default)
	DeleteReplyTo bool          // delete message what bot replays to
}

// HTTPClient wrap http.Client to allow mocking
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// SuperUser defines interface checking ig user name in su list
type SuperUser interface {
	IsSuper(userName string) bool
}

// SenderChat is the sender of the message, sent on behalf of a chat. The
// channel itself for channel messages. The supergroup itself for messages
// from anonymous group administrators. The linked channel for messages
// automatically forwarded to the discussion group
type SenderChat struct {
	// ID is a unique identifier for this chat
	ID int64 `json:"id"`
	// field below used only for logging purposes
	// UserName for private chats, supergroups and channels if available, optional
	UserName string `json:"username,omitempty"`
}

// Message is primary record to pass data from/to bots
type Message struct {
	ID         int
	From       User
	SenderChat SenderChat `json:"sender_chat,omitempty"`
	ChatID     int64
	Sent       time.Time
	HTML       string    `json:",omitempty"`
	Text       string    `json:",omitempty"`
	Entities   *[]Entity `json:",omitempty"`
	Image      *Image    `json:",omitempty"`
	ReplyTo    struct {
		From       User
		Text       string `json:",omitempty"`
		Sent       time.Time
		SenderChat SenderChat `json:"sender_chat,omitempty"`
	} `json:",omitempty"`
}

// Entity represents one special entity in a text message.
// For example, hashtags, usernames, URLs, etc.
type Entity struct {
	Type   string
	Offset int
	Length int
	URL    string `json:",omitempty"` // for “text_link” only, url that will be opened after user taps on the text
	User   *User  `json:",omitempty"` // for “text_mention” only, the mentioned user
}

// Image represents image
type Image struct {
	// fileID corresponds to Telegram file_id
	FileID   string
	Width    int
	Height   int
	Caption  string    `json:",omitempty"`
	Entities *[]Entity `json:",omitempty"`
}

// User defines user info of the Message
type User struct {
	ID          int64
	Username    string
	DisplayName string
}

// MultiBot combines many bots to one virtual
type MultiBot []Interface

// Help returns help message
func (b MultiBot) Help() string {
	sb := strings.Builder{}
	for _, child := range b {
		help := child.Help()
		if help != "" {
			// WriteString always returns nil err
			if !strings.HasSuffix(help, "\n") {
				help += "\n"
			}
			_, _ = sb.WriteString(help)
		}
	}
	return sb.String()
}

// OnMessage pass msg to all bots and collects responses (combining all of them)
// noinspection GoShadowedVar
func (b MultiBot) OnMessage(msg Message) (response Response) {
	if contains([]string{"help", "/help", "help!"}, msg.Text) {
		return Response{
			Text: b.Help(),
			Send: true,
		}
	}

	resps := make(chan string)
	var pin, unpin int32
	var channelID int64
	var banInterval time.Duration
	var user User
	var mutex = &sync.Mutex{}
	var replyTo int
	var deleteReplyTo int32

	wg := syncs.NewSizedGroup(4)
	for _, bot := range b {
		wg.Go(func(context.Context) {
			resp := bot.OnMessage(msg)
			if !resp.Send {
				return
			}

			resps <- resp.Text

			// update flags based on response
			if resp.Pin {
				atomic.AddInt32(&pin, 1)
			}
			if resp.Unpin {
				atomic.AddInt32(&unpin, 1)
			}
			if resp.ReplyTo > 0 {
				replyTo = resp.ReplyTo
			}

			// handle ban information
			if resp.BanInterval > 0 {
				mutex.Lock()
				if resp.BanInterval > banInterval {
					banInterval = resp.BanInterval
				}
				user = resp.User
				channelID = resp.ChannelID
				mutex.Unlock()
			}

			if resp.DeleteReplyTo {
				atomic.AddInt32(&deleteReplyTo, 1)
			}
		})
	}

	go func() {
		wg.Wait()
		close(resps)
	}()

	lines := make([]string, 0, len(resps))
	for r := range resps {
		log.Printf("[DEBUG] collect %q", r)
		lines = append(lines, r)
	}

	sort.Slice(lines, func(i, j int) bool {
		return lines[i] < lines[j]
	})

	if len(lines) > 0 {
		log.Printf("[DEBUG] answers %d, send %v", len(lines), len(lines) > 0)
	}
	return Response{
		Text:          strings.Join(lines, "\n"),
		Send:          len(lines) > 0,
		Pin:           atomic.LoadInt32(&pin) > 0,
		Unpin:         atomic.LoadInt32(&unpin) > 0,
		BanInterval:   banInterval,
		User:          user,
		ChannelID:     channelID,
		ReplyTo:       replyTo,
		DeleteReplyTo: atomic.LoadInt32(&deleteReplyTo) > 0,
	}
}

// ReactOn returns combined list of all keywords
func (b MultiBot) ReactOn() (res []string) {
	for _, bot := range b {
		res = append(res, bot.ReactOn()...)
	}
	return res
}

func contains(s []string, e string) bool {
	e = strings.TrimSpace(e)
	for _, a := range s {
		if strings.EqualFold(a, e) {
			return true
		}
	}
	return false
}

func makeHTTPRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to make request %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// EscapeMarkDownV1Text escapes markdownV1 special characters, used in places where we want to send text as-is.
// For example, telegram username with underscores would be italicized if we don't escape it.
// https://core.telegram.org/bots/api#markdown-style
func EscapeMarkDownV1Text(text string) string {
	escSymbols := []string{"_", "*", "`", "["}
	for _, esc := range escSymbols {
		text = strings.ReplaceAll(text, esc, "\\"+esc)
	}
	return text
}

// DisplayName returns user's display name or username or id
func DisplayName(msg Message) string {
	displayUsername := msg.From.DisplayName
	if displayUsername == "" {
		displayUsername = msg.From.Username
	}
	if displayUsername == "" {
		displayUsername = fmt.Sprintf("%d", msg.From.ID)
	}
	return displayUsername
}
