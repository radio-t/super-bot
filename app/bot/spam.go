package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type SpamFilter struct {
	casAPI string
	dry    bool
	client HTTPClient

	approvedUsers map[int64]bool
}

// If user is restricted for more than 366 days or less than 30 seconds from the current time,
// they are considered to be restricted forever.
var permanentBanDuration = time.Hour * 24 * 400

// NewSpamFilter makes a spam detecting bot
func NewSpamFilter(api string, client HTTPClient, dry bool) *SpamFilter {
	log.Printf("[INFO] Excerpt bot with %s", api)
	return &SpamFilter{casAPI: api, client: client, dry: dry, approvedUsers: map[int64]bool{}}
}

func (s *SpamFilter) OnMessage(msg Message) (response Response) {
	if s.approvedUsers[msg.From.ID] {
		return Response{}
	}

	reqURL := fmt.Sprintf("%s/check?user_id=%d", s.casAPI, msg.From.ID)
	req, err := http.NewRequest("GET", reqURL, http.NoBody)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return Response{}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return Response{}
	}
	defer resp.Body.Close()

	respData := struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		log.Printf("[WARN] failed to parse response from %s, error=%v", reqURL, err)
		return Response{}
	}
	if respData.OK {
		log.Printf("[INFO] user %s is not a spammer, added to aproved", msg.From.Username)
		s.approvedUsers[msg.From.ID] = true
		return Response{}
	}

	log.Printf("[INFO] user %s detected as spammer: %s, msg: %s", msg.From.Username, respData.Description, msg.Text)
	if s.dry {
		return Response{Text: "this is spam, but I'm in dry mode, so I'll do nothing yet", Send: true, ReplyTo: msg.ID}
	}
	return Response{Text: "this is spam, go to ban " + msg.From.DisplayName, Send: true, ReplyTo: msg.ID, BanInterval: permanentBanDuration, DeleteReplyTo: true}
}

// Help returns help message
func (s *SpamFilter) Help() string { return "" }

// ReactOn keys
func (s *SpamFilter) ReactOn() []string { return []string{} }
