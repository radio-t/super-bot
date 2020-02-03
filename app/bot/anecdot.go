package bot

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	log "github.com/go-pkgz/lgr"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// Anecdote bot, returns from http://rzhunemogu.ru/RandJSON.aspx?CType=1
type Anecdote struct {
	client HTTPClient
}

// NewAnecdote makes a bot for http://rzhunemogu.ru
func NewAnecdote(client HTTPClient) *Anecdote {
	log.Printf("[INFO] anecdote bot with http://rzhunemogu.ru/RandJSON.aspx?CType=1 and http://api.icndb.com/jokes/random")
	return &Anecdote{client: client}
}

// OnMessage returns one entry
func (a Anecdote) OnMessage(msg Message) (response Response) {

	if !contains(a.ReactOn(), msg.Text) {
		return Response{}
	}

	if contains([]string{"chuck!", "/chuck"}, msg.Text) {
		return a.chuck()
	}

	return a.rzhunemogu()
}

func (a Anecdote) rzhunemogu() (response Response) {
	reqURL := "http://rzhunemogu.ru/RandJSON.aspx?CType=1"

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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[WARN] failed to read body, error=%v", err)
		return Response{}
	}

	text := string(body)
	// this json is not really json? body with \r
	text = strings.TrimPrefix(text, `{"content":"`)
	text = strings.TrimSuffix(text, `"}`)

	tr := transform.NewReader(strings.NewReader(text), charmap.Windows1251.NewDecoder())
	buf, err := ioutil.ReadAll(tr)
	if err != nil {
		log.Printf("[WARN] failed to convert string to utf, error=%v", err)
		return Response{}
	}

	return Response{Text: string(buf), Send: true}
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
	return []string{"анекдот!", "анкедот!", "joke!", "chuck!", "/анекдот", "/joke", "/chuck"}
}
