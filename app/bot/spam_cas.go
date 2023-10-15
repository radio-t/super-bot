package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// SpamCasFilter bot, checks if user is a spammer using CAS API
type SpamCasFilter struct {
	casAPI    string
	dry       bool
	client    HTTPClient
	superUser SuperUser

	approvedUsers map[int64]bool
}

// If user is restricted for more than 366 days or less than 30 seconds from the current time,
// they are considered to be restricted forever.
var permanentBanDuration = time.Hour * 24 * 400

// NewSpamCasFilter makes a spam detecting bot
func NewSpamCasFilter(api string, client HTTPClient, superUser SuperUser, dry bool) *SpamCasFilter {
	log.Printf("[INFO] Spam bot (cas) with %s", api)
	return &SpamCasFilter{casAPI: api, client: client, dry: dry, approvedUsers: map[int64]bool{}, superUser: superUser}
}

// OnMessage checks if user already approved and if not checks if user is a spammer
func (s *SpamCasFilter) OnMessage(msg Message) (response Response) {
	if s.approvedUsers[msg.From.ID] || msg.From.ID == 0 {
		return Response{}
	}

	if s.superUser.IsSuper(msg.From.Username) {
		return Response{} // don't check super users for spam
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
		OK          bool   `json:"ok"` // ok means user is a spammer
		Description string `json:"description"`
	}{}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		log.Printf("[WARN] failed to parse response from %s, error=%v", reqURL, err)
		return Response{}
	}
	log.Printf("[DEBUG] response from %s: %+v", reqURL, respData)
	displayUsername := DisplayName(msg)
	if !respData.OK {
		log.Printf("[INFO] user %q is not a spammer, added to aproved", displayUsername)
		s.approvedUsers[msg.From.ID] = true
		return Response{}
	}

	log.Printf("[INFO] user %q detected as spammer: %s, msg: %q", displayUsername, respData.Description, msg.Text)
	if s.dry {
		return Response{
			Text: fmt.Sprintf("this is spam from %q, but I'm in dry mode, so I'll do nothing yet", displayUsername),
			Send: true, ReplyTo: msg.ID,
		}
	}
	return Response{Text: "this is spam! go to ban, " + displayUsername, Send: true, ReplyTo: msg.ID,
		BanInterval: permanentBanDuration, DeleteReplyTo: true}
}

// Help returns help message
func (s *SpamCasFilter) Help() string { return "" }

// ReactOn keys
func (s *SpamCasFilter) ReactOn() []string { return []string{} }
