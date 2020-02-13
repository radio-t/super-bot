package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "github.com/go-pkgz/lgr"
	"github.com/pkg/errors"
)

// PrepPost bot notifies on new prep topic on the site
type PrepPost struct {
	client        HTTPClient
	siteAPI       string
	checkDuration time.Duration

	last struct {
		prepPost postInfo
		checked  time.Time
	}
}

type postInfo struct {
	URL        string   `json:"url"`
	Title      string   `json:"title"`
	Categories []string `json:"categories"`
}

var errNotPost = errors.New("not prep post")

// NewPrepPost makes new PrepPost bot sending and pinning message "Начался сбор тем"
func NewPrepPost(client HTTPClient, api string, d time.Duration) *PrepPost {
	log.Printf("[INFO] prep-post bot with api %s", api)
	return &PrepPost{client: client, siteAPI: api, checkDuration: d}
}

// OnMessage reacts on any message and, from time to time (every checkDuration) hits site api
// and gets the latest prep article. In case if article's url changed returns pinned response.
// Skips the first check to avoid false-positive on restart
func (p *PrepPost) OnMessage(Message) (response Response) {

	if time.Now().Sub(p.last.checked) < p.checkDuration {
		return Response{}
	}

	defer func() {
		p.last.checked = time.Now()
	}()

	pi, err := p.recentPrepPost()
	if err != nil {
		if err != errNotPost {
			log.Printf("[WARN] failed to check for new post, %v", err)
		}
		return Response{}
	}

	defer func() {
		p.last.prepPost = pi
	}()

	if p.last.prepPost.URL != "" && pi.URL != p.last.prepPost.URL {
		return Response{Send: true, Pin: true, Text: fmt.Sprintf("Сбор тем начался - %s", pi.URL)}
	}
	return Response{}
}

func (p *PrepPost) recentPrepPost() (pi postInfo, err error) {

	reqURL := fmt.Sprintf("%s/last/1?categories=prep", p.siteAPI)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return pi, errors.Wrapf(err, "failed to make request %s", reqURL)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return pi, errors.Wrapf(err, "failed to send request %s", reqURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return pi, errors.Errorf("request %s returned %d", reqURL, resp.StatusCode)
	}

	posts := []postInfo{}
	if err := json.NewDecoder(resp.Body).Decode(&posts); err != nil {
		return pi, errors.Wrapf(err, "failed to parse response from %s", reqURL)
	}

	if len(posts) > 0 {
		return posts[0], nil
	}
	return pi, errNotPost
}

// Help returns help message
func (p *PrepPost) Help() string {
	return ""
}

// ReactOn keys
func (p *PrepPost) ReactOn() []string {
	return []string{}
}
