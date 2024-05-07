package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// News bot, returns numArticles last articles in MD format from https://news.radio-t.com/api/v1/news/lastmd/5
type News struct {
	client      HTTPClient
	newsAPI     string
	numArticles int
}

type newsArticle struct {
	Title string    `json:"title"`
	Link  string    `json:"link"`
	Ts    time.Time `json:"ats"` // nolint
}

// NewNews makes new News bot
func NewNews(client HTTPClient, api string, maximum int) *News {
	log.Printf("[INFO] news bot with api %s", api)
	return &News{client: client, newsAPI: api, numArticles: maximum}
}

// Help returns help message
func (n News) Help() string {
	return GenHelpMsg(n.ReactOn(), "5 последних новостей для Радио-Т")
}

// OnMessage returns N last news articles
func (n News) OnMessage(msg Message) (response Response) {
	if !contains(n.ReactOn(), msg.Text) {
		return Response{}
	}

	reqURL := fmt.Sprintf("%s/v1/news/last/%d", n.newsAPI, n.numArticles)
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
	defer resp.Body.Close() // nolint

	articles := []newsArticle{}
	if err = json.NewDecoder(resp.Body).Decode(&articles); err != nil {
		log.Printf("[WARN] failed to parse response, error %v", err)
		return Response{}
	}

	lines := make([]string, 0, len(articles))
	for _, a := range articles {
		if a.Title == "" {
			a.Title = "безымянная новость"
		}
		lines = append(lines, fmt.Sprintf("- [%s](%s) %s", a.Title, a.Link, a.Ts.Format("2006-01-02")))
	}
	return Response{
		Text: strings.Join(lines, "\n") + "\n- [все новости и темы](https://news.radio-t.com)",
		Send: true,
	}
}

// ReactOn keys
func (n News) ReactOn() []string {
	return []string{"news!", "новости!"}
}
