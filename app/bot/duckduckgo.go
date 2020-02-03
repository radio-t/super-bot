package bot

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/go-pkgz/lgr"
)

// Duck bot, returns from duckduckgo via mashape
type Duck struct {
	mashapeKey string
	client     HTTPClient
}

// NewDuck makes a bot for duckduckgo
func NewDuck(key string, client HTTPClient) *Duck {
	log.Printf("[INFO] Duck bot with duckduckgo-duckduckgo-zero-click-info.p.mashape.com")
	return &Duck{mashapeKey: key, client: client}
}

// OnMessage pass msg to all bots and collects responses
func (d *Duck) OnMessage(msg Message) (response Response) {

	ok, reqText := d.request(msg.Text)
	if !ok {
		return Response{}
	}

	reqURL := fmt.Sprintf("https://duckduckgo-duckduckgo-zero-click-info.p.mashape.com/?format=json&no_html=1&no_redirect=1&q=%s&skip_disambig=1", reqText)

	req, err := makeHTTPRequest(reqURL)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return Response{}
	}
	req.Header.Set("X-Mashape-Key", d.mashapeKey)
	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return Response{}
	}
	defer func() { _ = resp.Body.Close() }()

	duckResp := struct {
		AbstractText   string `json:"AbstractText"`
		AbstractSource string `json:"AbstractSource"`
		AbstractURL    string `json:"AbstractURL"`
		Image          string `json:"Image"`
	}{}
	if err = json.NewDecoder(resp.Body).Decode(&duckResp); err != nil {
		log.Printf("[WARN] failed to convert from json, error=%v", err)
		return Response{}
	}

	mdLink := func(inp string) string {
		result := strings.Replace(inp, "(", "%28", -1)
		result = strings.Replace(result, ")", "%29", -1)
		return result
	}

	if duckResp.AbstractText == "" {
		return Response{
			Text: fmt.Sprintf("_не в силах. но могу помочь_ [это поискать](https://duckduckgo.com/?q=%s)", mdLink(reqText)),
			Send: true,
		}
	}

	respMD := fmt.Sprintf("- %s\n[%s](%s)", duckResp.AbstractText, duckResp.AbstractSource, mdLink(duckResp.AbstractURL))
	return Response{
		Text: respMD,
		Send: true,
	}
}

func (d *Duck) request(text string) (react bool, reqText string) {

	for _, prefix := range d.ReactOn() {
		if strings.HasPrefix(text, prefix) {
			return true, strings.Replace(strings.TrimSpace(strings.TrimPrefix(text, prefix)), " ", "+", -1)
		}
	}
	return false, ""
}

// ReactOn keys
func (d *Duck) ReactOn() []string {
	return []string{"ddg!", "??", "/ddg"}
}
