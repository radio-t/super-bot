package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
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
	Body       string    `json:"body"`
	ShowNum    int       `json:"show_num,omitempty"`
}

// NewPodcasts makes new Podcasts bot
func NewPodcasts(client HTTPClient, api string, maxResults int) *Podcasts {
	log.Printf("[INFO] podcasts bot with api %s", api)
	return &Podcasts{client: client, siteAPI: api, maxResults: maxResults}
}

// OnMessage returns result of search via https://radio-t.com/site-api/search?
func (p *Podcasts) OnMessage(msg Message) (response Response) {

	defer func() { // to catch possible panics from potentially dangerous makeBotResponse
		if r := recover(); r != nil {
			response.Text = ""
			response.Send = false
		}
	}()

	ok, reqText := p.request(msg.Text)
	if !ok {
		return Response{}
	}

	reqURL := fmt.Sprintf("%s/search?limit=%d&q=%s", p.siteAPI, p.maxResults, reqText)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return Response{}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return Response{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[WARN] request %s returned %s", reqURL, resp.Status)
		return Response{}
	}

	sr := []siteAPIResp{}
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		log.Printf("[WARN] failed to parse response from %s, error=%v", reqURL, err)
		return Response{}
	}
	return Response{
		Text: p.makeBotResponse(sr, reqText),
		Send: true,
	}
}

func (p *Podcasts) makeBotResponse(sr []siteAPIResp, reqText string) string {

	makeRepLine := func(nl noteWithLink) string {
		if nl.link != "" {
			return fmt.Sprintf("[%s](%s)", nl.text, nl.link)
		}
		return nl.text
	}

	if len(sr) == 0 {
		return fmt.Sprintf("ничего не нашел на запрос %q", reqText)
	}

	var res string
	for _, s := range sr {
		nls := p.notesWithLinks(s)
		res += fmt.Sprintf("[Радио-Т #%d](%s) _%s_\n", s.ShowNum, s.URL, s.Date.Format("02 Jan 06"))
		for _, nl := range nls {

			if strings.Contains(strings.ToLower(nl.text), strings.ToLower(reqText)) {
				res += "●  " + makeRepLine(nl) + "\n"
				continue
			}

			if strings.Contains(strings.ToLower(nl.link), strings.ToLower(reqText)) {
				res += "○  " + makeRepLine(nl) + "\n"
				continue
			}
		}
		res += "\n"
	}
	return res
}

type noteWithLink struct {
	text string
	link string
}

// linRe is a matching regex for links from href
var linkRe = regexp.MustCompile(`<a\s+(?:[^>]*?\s+)?href="([^"]*)"`)

// notesWithLinks gets notes and matching links from body
func (p *Podcasts) notesWithLinks(s siteAPIResp) (res []noteWithLink) {

	// show notes may start with multiple \n, strip them all
	showNotes := s.ShowNotes
	for i := 0; i < 10; i++ {
		showNotes = strings.TrimPrefix(showNotes, "\n")
	}

	notes := strings.Split(showNotes, "\n")
	links := strings.Split(s.Body, "<li>")[1:] // we don't care what the 0 element was, links starts with <li>
	for i, note := range notes {
		if strings.Contains(note, " лог чата") || strings.Contains(note, "Темы наших слушателей") {
			break
		}
		nl := noteWithLink{text: note}
		if i < len(links) {
			ll := linkRe.FindStringSubmatch(links[i])
			if len(ll) >= 2 {
				nl.link = ll[1]
			}
		}
		res = append(res, nl)
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
