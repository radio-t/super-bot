package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Excerpt bot, returns link excerpt
type Excerpt struct {
	api   string
	token string
}

var (
	rLink = regexp.MustCompile(`(https?://[a-zA-Z0-9\-.]+\.[a-zA-Z]{2,3}(/\S*)?)`)
	rImg  = regexp.MustCompile(`\.gif|\.jpg|\.jpeg|\.png`)
)

// NewExcerpt makes a bot extracting articles excerpt
func NewExcerpt(api, token string) *Excerpt {
	log.Printf("[INFO] Excerpt bot with %s", api)
	return &Excerpt{api: api, token: token}
}

// Help returns help message
func (e *Excerpt) Help() string {
	return ""
}

// OnMessage pass msg to all bots and collects responses
func (e *Excerpt) OnMessage(msg Message) (response Response) {

	link, err := e.link(msg.Text)
	if err != nil {
		return Response{}
	}

	client := http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("%s?token=%s&url=%s", e.api, e.token, link)
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[WARN] can't send request to parse article to %s, %v", url, err)
		return Response{}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		log.Printf("[WARN] parser error code %d for %v", resp.StatusCode, url)
		return Response{}
	}

	r := struct {
		Title   string `json:"title"`
		Excerpt string `json:"excerpt"`
	}{}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[WARN] can't read response for %s, %v", url, err)
		return Response{}
	}

	if err := json.Unmarshal(body, &r); err != nil {
		log.Printf("[WARN] can't decode response for %s, %v", url, err)
	}

	return Response{
		Text: fmt.Sprintf("%s\n\n_%s_", r.Excerpt, r.Title),
		Send: true,
	}
}

func (e *Excerpt) link(input string) (link string, err error) {

	if strings.Contains(input, "twitter.com") {
		log.Printf("ignore possible twitter link from %s", input)
		return "", fmt.Errorf("ignore twitter")
	}

	if l := rLink.FindString(input); l != "" && !rImg.MatchString(l) {
		log.Printf("found a link %s", l)
		return l, nil
	}
	return "", fmt.Errorf("no link found")
}

// ReactOn keys
func (e *Excerpt) ReactOn() []string {
	return []string{}
}
