package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

// Podcasts search bot, returns search result via site-api see https://radio-t.com/api-docs/
// GET /search?q=text-to-search&skip=10&limit=5, example: : https://radio-t.com/site-api/search?q=mongo&limit=10
type Podcasts struct {
	client     HTTPClient
	siteAPI    string
	maxResults int
}

type siteAPIResp struct {
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	Date       time.Time `json:"date"`
	Categories []string  `json:"categories"`
	Image      string    `json:"image,omitempty"`
	FileName   string    `json:"file_name,omitempty"`
	ShowNotes  string    `json:"show_notes,omitempty"`
	ShowNum    int       `json:"show_num,omitempty"`
}

// NewPodcasts makes new Podcasts bot
func NewPodcasts(client HTTPClient, api string, maxResults int) *Podcasts {
	log.Printf("[INFO] podcasts bot with api %s", api)
	return &Podcasts{client: client, siteAPI: api, maxResults: maxResults}
}

// OnMessage returns 5 last news articles
func (p *Podcasts) OnMessage(msg Message) (response string, answer bool) {

	ok, reqText := p.request(msg.Text)
	if !ok {
		return "", false
	}

	reqURL := fmt.Sprintf("%s/search?limit=%d&q=%s", p.siteAPI, p.maxResults, reqText)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return "", false
	}

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WARN] request %s returned %s", reqURL, resp.Status)
		return "", false
	}

	sr := []siteAPIResp{}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		log.Printf("[WARN] failed to parse response from %s, error=%v", reqURL, err)
		return "", false
	}
	return p.makeBotResponse(sr, reqText), true
}

func (p *Podcasts) makeBotResponse(sr []siteAPIResp, reqText string) string {
	if len(sr) == 0 {
		return fmt.Sprintf("ничего не нашел на запрос %q", reqText)
	}
	var res string
	for _, s := range sr {
		res += fmt.Sprintf("[Радио-Т #%d](%s) _%s_\n\n", s.ShowNum, s.URL, s.Date.Format("02 Jan 06"))
		for _, s := range strings.Split(s.ShowNotes, "\n") {
			if len(s) < 2 || strings.Contains(s, " лог чата") {
				continue
			}
			res += fmt.Sprintf("- %s\n", s)
		}
		res += "\n"

	}
	return res
}

func (p *Podcasts) request(text string) (react bool, reqText string) {

	for _, prefix := range p.ReactOn() {
		if strings.HasPrefix(text, prefix) {
			return true, strings.Replace(strings.TrimSpace(strings.TrimPrefix(text, prefix)), " ", "+", -1)
		}
	}
	return false, ""
}

// ReactOn keys
func (p *Podcasts) ReactOn() []string {
	return []string{"search!", "/search"}
}
