package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

// News bot, returns 5 last articles in MD from https://news.radio-t.com/api/v1/news/lastmd/5
type News struct {
	client  HTTPClient
	newsAPI string
}

type newsArticle struct {
	Title string    `json:"title"`
	Link  string    `json:"link"`
	Ts    time.Time `json:"ats"`
}

// NewNews makes new News bot
func NewNews(client HTTPClient, api string) *News {
	log.Printf("[INFO] news bot with api %s", api)
	return &News{client: client, newsAPI: api}
}

// OnMessage returns 5 last news articles
func (n News) OnMessage(msg Message) (response Response) {
	if !contains(n.ReactOn(), msg.Text) {
		return Response{}
	}

	reqURL := fmt.Sprintf("%s/v1/news/last/5", n.newsAPI)
	log.Printf("[DEBUG] request %s", reqURL)

	req, err := makeHTTPRequest(reqURL)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return Response{}
	}

	resp, err := n.client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return Response{}
	}
	defer resp.Body.Close()

	articles := []newsArticle{}
	if err = json.NewDecoder(resp.Body).Decode(&articles); err != nil {
		log.Printf("[WARN] failed to parse response, error %v", err)
		return Response{}
	}

	var lines []string
	for _, a := range articles {
		lines = append(lines, fmt.Sprintf("- [%s](%s) %s", a.Title, a.Link, a.Ts.Format("2006-01-02")))
	}
	return Response{
		Text: strings.Join(lines, "\n") + "\n- [все новости и темы](https://news.radio-t.com)",
		Send: true,
	}
}

// ReactOn keys
func (n News) ReactOn() []string {
	return []string{"news!", "новости!", "/news", "/новости"}
}
