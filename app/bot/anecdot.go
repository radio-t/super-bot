package bot

import (
	"encoding/json"
	"log"
	"strings"
)

// Anecdote bot, returns from https://jokesrv.rubedo.cloud/
type Anecdote struct {
	client HTTPClient
}

// NewAnecdote makes a bot for http://rzhunemogu.ru
func NewAnecdote(client HTTPClient) *Anecdote {
	log.Printf("[INFO] anecdote bot with https://jokesrv.rubedo.cloud/ and http://api.icndb.com/jokes/random")
	return &Anecdote{client: client}
}

// Help returns help message
func (a Anecdote) Help() string {
	return genHelpMsg(a.ReactOn(), "расскажет анекдот или шутку")
}

// OnMessage returns one entry
func (a Anecdote) OnMessage(msg Message) (response Response) {

	if !contains(a.ReactOn(), msg.Text) {
		return Response{}
	}

	if contains([]string{"chuck!", "/chuck"}, msg.Text) {
		return a.chuck()
	}

	switch {
	case contains([]string{"chuck!", "/chuck"}, msg.Text):
		return a.chuck()
	case contains([]string{"facts!", "/facts"}, msg.Text):
		return a.jokesrv("facts")
	case contains([]string{"zaibatsu!", "/zaibatsu"}, msg.Text):
		return a.jokesrv("zaibatsu")
	case contains([]string{"excuse!", "/excuse"}, msg.Text):
		return a.jokesrv("excuse")
	default:
		return a.jokesrv("oneliner")
	}

}

func (a Anecdote) jokesrv(category string) (response Response) {
	reqURL := "https://jokesrv.rubedo.cloud/" + category

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
	defer resp.Body.Close()
	rr := struct {
		Category string `json:"category"`
		Content  string `json:"content"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		log.Printf("[WARN] failed to parse body, error=%v", err)
		return Response{}

	}

	return Response{Text: rr.Content, Send: true}
}

func (a Anecdote) chuck() (response Response) {

	chuckResp := struct {
		Type  string
		Value struct {
			Categories []string
			Joke       string
		}
	}{}

	reqURL := "http://api.icndb.com/jokes/random"
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
	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&chuckResp); err != nil {
		log.Printf("[WARN] failed to convert from json, error=%v", err)
		return Response{}
	}
	return Response{
		Text: "- " + strings.Replace(chuckResp.Value.Joke, "&quot;", "\"", -1),
		Send: true,
	}
}

// ReactOn keys
func (a Anecdote) ReactOn() []string {
	return []string{"анекдот!", "анкедот!", "joke!", "chuck!", "facts!", "zaibatsu!", "excuse!"}
}
