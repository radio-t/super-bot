package bot

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

// StackOverflow bot, returns from "https://api.stackexchange.com/2.2/questions?order=desc&sort=activity&site=stackoverflow"
// reacts on "so!" prefix, i.e. "so! golang"
type StackOverflow struct{}

// StackOverflow for json response
type soResponse struct {
	Items []struct {
		Title string   `json:"title"`
		Link  string   `json:"link"`
		Tags  []string `json:"tags"`
	} `json:"items"`
}

// NewStackOverflow makes a bot for SO
func NewStackOverflow() *StackOverflow {
	log.Printf("[INFO] StackOverflow bot with https://api.stackexchange.com/2.2/questions")
	return &StackOverflow{}
}

// OnMessage returns one entry
func (s StackOverflow) OnMessage(msg Message) (response Response) {

	if !contains(s.ReactOn(), msg.Text) {
		return Response{}
	}

	reqURL := "https://api.stackexchange.com/2.2/questions?order=desc&sort=activity&site=stackoverflow"
	client := http.Client{Timeout: 5 * time.Second}

	req, err := makeHTTPRequest(reqURL)
	if err != nil {
		log.Printf("[WARN] failed to prep request %s, error=%v", reqURL, err)
		return Response{}
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return Response{}
	}
	defer resp.Body.Close()

	soRecs := soResponse{}

	if err := json.NewDecoder(resp.Body).Decode(&soRecs); err != nil {
		log.Printf("[WARN] failed to parse response, error %v", err)
		return Response{}
	}
	if len(soRecs.Items) == 0 {
		return Response{}
	}

	r := soRecs.Items[rand.Intn(len(soRecs.Items))]
	return Response{
		Text: fmt.Sprintf("[%s](%s) %s", r.Title, r.Link, strings.Join(r.Tags, ",")),
		Send: true,
	}
}

// ReactOn keys
func (s StackOverflow) ReactOn() []string {
	return []string{"so!", "/so"}
}
