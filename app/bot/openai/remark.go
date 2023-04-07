package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-pkgz/notify"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// remarkComment is json structure for a comment from Remark42
type remarkComment struct {
	ParentID string `json:"pid"`
	Text     string `json:"text"`
	User     struct {
		Name     string `json:"name"`
		Admin    bool   `json:"admin"`
		Verified bool   `json:"verified,omitempty"`
	} `json:"user"`
	Score     int       `json:"score"`
	Deleted   bool      `json:"delete,omitempty" bson:"delete"`
	Timestamp time.Time `json:"time" bson:"time"`
}

// render returns a string representation of the comment
func (c remarkComment) render() string {
	user := tbapi.EscapeText(tbapi.ModeHTML, c.User.Name)
	text := notify.TelegramSupportedHTML(c.Text)
	return fmt.Sprintf("<b>%+d</b> от <b>%s</b>\n<i>%s</i>", c.Score, user, text)
}

// getLink returns a first link from the comment text
func (c remarkComment) getLink() string {
	// Find only links in the comment
	reLink := regexp.MustCompile(`href="(https?://[^\s"'<>]+)"`)

	parts := reLink.FindStringSubmatch(c.Text)

	if len(parts) < 2 {
		return ""
	}

	link := parts[1]
	if link == "" || strings.Contains(link, "radio-t.com") {
		return ""
	}

	return link
}

// RemarkClient is a client for Remark42
type RemarkClient struct {
	*http.Client
	API string
}

func (c RemarkClient) getComments(remarkLink string) (comments []remarkComment, err error) {
	rl := fmt.Sprintf("%s?site=radiot&url=%s&sort-score&format=plain", c.API, remarkLink)
	resp, err := c.Get(rl)
	if err != nil {
		return []remarkComment{}, fmt.Errorf("can't get comments for %s: %w", remarkLink, err)
	}
	defer resp.Body.Close() // nolint
	if resp.StatusCode != http.StatusOK {
		return []remarkComment{}, fmt.Errorf("can't get comments for %s: %d", remarkLink, resp.StatusCode)
	}

	urResp := struct {
		Comments []remarkComment `json:"comments"`
	}{}

	if decErr := json.NewDecoder(resp.Body).Decode(&urResp); decErr != nil {
		return []remarkComment{}, fmt.Errorf("can't decode comments for %s: %w", remarkLink, decErr)
	}

	for _, c := range urResp.Comments {
		if c.ParentID != "" || c.Deleted {
			continue
		}

		comments = append(comments, c)
	}
	return comments, nil
}

// GetTopComments returns top comments for the remark link sorted by score and time
func (c RemarkClient) GetTopComments(remarkLink string) (comments, links []string, err error) {
	remarkComments, err := c.getComments(remarkLink)
	if err != nil {
		return comments, links, err
	}

	positiveComments := make([]remarkComment, 0, len(remarkComments))
	for _, c := range remarkComments {
		if c.Score >= 0 {
			positiveComments = append(positiveComments, c)
		}
	}

	sort.Slice(positiveComments, func(i, j int) bool {
		if positiveComments[i].Score < positiveComments[j].Score {
			return false
		}

		if positiveComments[i].Score > positiveComments[j].Score {
			return true
		}
		// Equal case
		return positiveComments[i].Timestamp.Before(positiveComments[j].Timestamp)
	})

	for _, c := range positiveComments {
		comments = append(comments, c.render())
		links = append(links, c.getLink())
	}

	return comments, links, nil
}
