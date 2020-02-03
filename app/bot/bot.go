package bot

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/go-pkgz/syncs"
	"github.com/pkg/errors"
)

//go:generate mockery -inpkg -name HTTPClient -case snake
//go:generate mockery -inpkg -name Interface -case snake
//go:generate mockery -inpkg -name SuperUser -case snake

// Interface is a bot reactive spec. response will be sent if "send" result is true
type Interface interface {
	OnMessage(msg Message) (response Response)
	ReactOn() []string
}

// Response describes bot's answer on particular message
type Response struct {
	Text string
	Send bool
	Pin  bool
}

// HTTPClient wrap http.Client to allow mocking
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// SuperUser defines interface checking ig user name in su list
type SuperUser interface {
	IsSuper(userName string) bool
}

// Message is primary record to pass data from/to bots
type Message struct {
	ID       int
	From     User
	Sent     time.Time
	HTML     string    `json:",omitempty"`
	Text     string    `json:",omitempty"`
	Entities *[]Entity `json:",omitempty"`
	Image    *Image    `json:",omitempty"`
}

// Entity represents one special entity in a text message.
// For example, hashtags, usernames, URLs, etc.
type Entity struct {
	Type   string
	Offset int
	Length int
	URL    string `json:",omitempty"` // For “text_link” only, url that will be opened after user taps on the text
	User   *User  `json:",omitempty"` // For “text_mention” only, the mentioned user
}

// Image represents image
type Image struct {
	// FileID corresponds to Telegram file_id
	FileID   string
	Width    int
	Height   int
	Caption  string    `json:",omitempty"`
	Entities *[]Entity `json:",omitempty"`
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
func (b MultiBot) OnMessage(msg Message) (response Response) {

	if contains([]string{"help", "/help", "help!"}, msg.Text) {
		return Response{
			Text: "_" + strings.Join(b.ReactOn(), " ") + "_",
			Send: true,
		}
	}

	resps := make(chan string)
	var pin int32 = 0

	wg := syncs.NewSizedGroup(4)
	for _, bot := range b {
		bot := bot
		wg.Go(func(ctx context.Context) {
			if resp := bot.OnMessage(msg); resp.Send {
				resps <- resp.Text
				if resp.Pin {
					atomic.AddInt32(&pin, 1)
				}
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
	return Response{
		Text: strings.Join(lines, "\n"),
		Send: len(lines) > 0,
		Pin:  atomic.LoadInt32(&pin) > 0,
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
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make request %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}
