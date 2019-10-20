package bot

import (
	"net/http"
	"strings"
	"sync"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
	"github.com/sromku/go-gitter"
)

// Interface is a bot reactive spec. response will be sent if "send" result is true
type Interface interface {
	OnMessage(msg gitter.Message) (response string, send bool)
	ReactOn() []string
}

type SuperUser interface {
	IsSuper(user gitter.User) bool
}

// MultiBot combines many bots to one virtual
type MultiBot []Interface

// OnMessage pass msg to all bots and collects reposnses (combining all of them)
func (b MultiBot) OnMessage(msg gitter.Message) (response string, send bool) {

	if contains([]string{"help", "/help", "help!"}, msg.Text) {
		return "_" + strings.Join(b.ReactOn(), " ") + "_", true
	}

	resps := make(chan string)
	var wg sync.WaitGroup
	wg.Add(len(b))
	for _, bot := range b {
		go func(bot Interface) {
			defer wg.Done()
			if resp, ok := bot.OnMessage(msg); ok {
				resps <- resp
			}
		}(bot)
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

func makeHttpRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make request %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return req, nil
}
