package bot

import (
	"context"
	"net/http"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/syncs"
	"github.com/pkg/errors"
)

// Interface is a bot reactive spec. response will be sent if "send" result is true
type Interface interface {
	OnMessage(msg Message) (response string, send bool)
	ReactOn() []string
}

// SuperUser defines interface checking ig user name in su list
type SuperUser interface {
	IsSuper(userName string) bool
}

// Message is primary record to pass data from/to bots
type Message struct {
	ID                   int
	From                 User
	Sent                 time.Time
	HTML                 string     `json:",omitempty"`
	Text                 string     `json:",omitempty"`
	Picture              *Picture   `json:",omitempty"`
	Document             *Document  `json:",omitempty"`
	Animation            *Animation `json:",omitempty"`
	Voice                *Voice     `json:",omitempty"`
	Video                *Video     `json:",omitempty"`
	ReplyToMessage       *Message   `json:",omitempty"`
	ForwardFrom          *User      `json:",omitempty"`
	ForwardFromChat      *Chat      `json:",omitempty"`
	ForwardFromMessageID int        `json:",omitempty"`
}

// Chat represents channel or chat
// Used in Message.ForwardFromChat
type Chat struct {
	ID        int64
	Type      string
	Title     string
	UserName  string
	FirstName string `json:",omitempty"`
	LastName  string `json:",omitempty"`
}

// Picture represents HTML5 <picture> with optional <figcaption>
type Picture struct {
	Image     Image
	Class     string   `json:",omitempty"`
	Caption   string   `json:",omitempty"`
	Sources   []Source `json:",omitempty"`
	Thumbnail *Source  `json:",omitempty"`
}

// Image represents HTML <img>
type Image struct {
	// FileID corresponds to Telegram file_id
	FileID  string
	Width   int
	Height  int
	Size    int
	Type    string   `json:",omitempty"`
	Alt     string   `json:",omitempty"`
	Sources []Source `json:",omitempty"`
}

// Source defines file, usually image
type Source struct {
	FileID string
	Width  int
	Height int
	Size   int    `json:",omitempty"`
	Type   string `json:",omitempty"`
}

// Document is for attached file on Telegram
type Document struct {
	FileID    string
	FileName  string
	Size      int
	MimeType  string
	Caption   string
	Thumbnail *Source
}

// Animation is for animation (MP4, GIF)
type Animation struct {
	FileID    string
	FileName  string
	Size      int
	MimeType  string
	Thumbnail *Source
	Duration  int
	Width     int
	Height    int
}

// Voice is for voice messages
type Voice struct {
	Duration int
	Sources  []Source
}

// Video is for video messages
type Video struct {
	FileID    string
	FileName  string
	Size      int
	MimeType  string
	Thumbnail *Source
	Duration  int
	Width     int
	Height    int
	Caption   string
}

// User defines user info of the Message
type User struct {
	ID          string `json:",omitempty"`
	Username    string
	DisplayName string
}

// MultiBot combines many bots to one virtual
type MultiBot []Interface

// OnMessage pass msg to all bots and collects reposnses (combining all of them)
//noinspection GoShadowedVar
func (b MultiBot) OnMessage(msg Message) (response string, send bool) {

	if contains([]string{"help", "/help", "help!"}, msg.Text) {
		return "_" + strings.Join(b.ReactOn(), " ") + "_", true
	}

	resps := make(chan string)

	wg := syncs.NewSizedGroup(4)
	for _, bot := range b {
		bot := bot
		wg.Go(func(ctx context.Context) {
			if resp, ok := bot.OnMessage(msg); ok {
				resps <- resp
			}
		})
	}

	go func() {
		wg.Wait()
		close(resps)
	}()

	var lines []string
	for r := range resps {
		log.Printf("[DEBUG] collect %q", r)
		lines = append(lines, r)
	}

	log.Printf("[DEBUG] answers %d, send %v", len(lines), len(lines) > 0)
	return strings.Join(lines, "\n"), len(lines) > 0
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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make request %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}
