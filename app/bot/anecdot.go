package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-pkgz/lcw"
)

// Anecdote bot, returns from jokesrv.fermyon.app or chucknorris.io
type Anecdote struct {
	client     HTTPClient
	categCache lcw.LoadingCache
}

// NewAnecdote makes a bot for jokesrv.fermyon.app and chucknorris.io
func NewAnecdote(client HTTPClient) *Anecdote {
	log.Printf("[INFO] anecdote bot with  https://jokesrv.fermyon.app and https://api.chucknorris.io/jokes/random")
	c, _ := lcw.NewExpirableCache(lcw.MaxKeys(100), lcw.TTL(time.Hour))
	return &Anecdote{client: client, categCache: c}
}

// Help returns help message
func (a Anecdote) Help() string {
	return GenHelpMsg(a.ReactOn(), "расскажет анекдот или шутку")
}

// OnMessage returns one entry
func (a Anecdote) OnMessage(msg Message) (response Response) {

	if !contains(a.ReactOn(), msg.Text) {
		return Response{}
	}

	if contains([]string{"chuck!", "/chuck"}, msg.Text) {
		return a.chuck()
	}

	cc, err := a.categories()
	if err != nil {
		log.Printf("[WARN] category retrival failed, %v", err)
	}

	switch {
	case contains([]string{"chuck!", "/chuck"}, msg.Text):
		return a.chuck()
	case contains(cc, msg.Text):
		return a.jokesrv(strings.TrimSuffix(strings.TrimPrefix(msg.Text, "/"), "!"))
	default:
		return a.jokesrv("oneliner")
	}

}

// get categorize from https://jokesrv.fermyon.app/categories and extend with / prefix and ! suffix
// to mach commands
func (a Anecdote) categories() ([]string, error) {
	res, err := a.categCache.Get("categories", func() (interface{}, error) {
		var categories []string
		req, err := http.NewRequest("GET", "https://jokesrv.fermyon.app/categories", http.NoBody)
		if err != nil {
			return nil, fmt.Errorf("can't make categories request: %w", err)
		}
		resp, err := a.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("can't send categories request: %w", err)
		}
		defer resp.Body.Close() // nolint
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad response code %d", resp.StatusCode)
		}
		err = json.NewDecoder(resp.Body).Decode(&categories)
		if err != nil {
			return nil, fmt.Errorf("can't decode category response: %w", err)
		}
		return categories, nil
	})
	if err != nil {
		return nil, err
	}

	cc := make([]string, 0, len(res.([]string)))
	for _, c := range res.([]string) {
		cc = append(cc, c+"!")
	}
	return cc, nil
}

func (a Anecdote) jokesrv(category string) (response Response) {
	reqURL := "https://jokesrv.fermyon.app/" + category

	req, err := makeHTTPRequest(reqURL)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return Response{}
	}
	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return Response{}
	}
	defer resp.Body.Close() // nolint
	rr := struct {
		Category string `json:"category"`
		Content  string `json:"content"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		log.Printf("[WARN] failed to parse body, error=%v", err)
		return Response{}

	}

	return Response{Text: EscapeMarkDownV1Text(strings.TrimSuffix(rr.Content, ".")), Send: true}
}

func (a Anecdote) chuck() (response Response) {

	chuckResp := struct {
		Value string
	}{}

	reqURL := "https://api.chucknorris.io/jokes/random"
	req, err := makeHTTPRequest(reqURL)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return Response{}
	}
	resp, err := a.client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return Response{}
	}
	defer resp.Body.Close() // nolint

	if err = json.NewDecoder(resp.Body).Decode(&chuckResp); err != nil {
		log.Printf("[WARN] failed to convert from json, error=%v", err)
		return Response{}
	}
	return Response{
		Text: EscapeMarkDownV1Text(chuckResp.Value),
		Send: true,
	}
}

// ReactOn keys
func (a Anecdote) ReactOn() []string {

	cc, err := a.categories()
	if err != nil {
		log.Printf("[WARN] category retrival failed, %v", err)
	}

	return append([]string{"анекдот!", "анкедот!", "joke!", "chuck!"}, cc...)
}
