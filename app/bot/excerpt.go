package bot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/sromku/go-gitter"
)

// Excerpt bot, returns link's excerpt
type Excerpt struct {
	api   string
	token string
}

var rLink = regexp.MustCompile(`(https?://[a-zA-Z0-9\-.]+\.[a-zA-Z]{2,3}(/\S*)?)`)
var rImg = regexp.MustCompile(`\.gif|\.jpg|\.jpeg|\.png`)

// NewExcerpt makes a bot extracting articles excerpt
func NewExcerpt(api string, token string) *Excerpt {
	log.Printf("[INFO] Excerpt bot with %s", api)
	return &Excerpt{api: api, token: token}
}

// OnMessage pass msg to all bots and collects responses
func (e *Excerpt) OnMessage(msg gitter.Message) (response string, answered bool) {

	link, err := e.link(msg.Text)
	if err != nil {
		return "", false
	}

	client := http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s?token=%s&url=%s", e.api, e.token, link)
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[WARN] can't send request to parse article to %s, %v", url, err)
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		log.Printf("[WARN] parser error code %d for %v", resp.StatusCode, url)
		return "", false
	}

	r := struct {
		Title   string `json:"title"`
		Excerpt string `json:"excerpt"`
	}{}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[WARN] can't read response for %s, %v", url, err)
		return "", false
	}

	if err := json.Unmarshal(body, &r); err != nil {
		log.Printf("[WARN] can't decode response for %s, %v", url, err)
	}

	return fmt.Sprintf("%s\n\n_%s_", r.Excerpt, r.Title), true
}

func (e *Excerpt) link(input string) (link string, err error) {

	if strings.Contains(input, "twitter.com") {
		log.Printf("ignore possible twitter link from %s", input)
		return "", errors.New("ignore twitter")
	}

	if link := rLink.FindString(input); link != "" && !rImg.MatchString(link) {
		log.Printf("found a link %s", link)
		return link, nil
	}
	return "", errors.New("no link found")
}

// ReactOn keys
func (e *Excerpt) ReactOn() []string {
	return []string{}
}
