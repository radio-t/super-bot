package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/go-pkgz/lgr"
)

// Duck bot, returns from duckduckgo via mashape
type Duck struct {
	mashapeKey string
}

// NewDuck makes a bot for duckduckgo
func NewDuck(key string) *Duck {
	log.Printf("[INFO] Duck bot with duckduckgo-duckduckgo-zero-click-info.p.mashape.com")
	return &Duck{mashapeKey: key}
}

// OnMessage pass msg to all bots and collects responses
func (d *Duck) OnMessage(msg Message) (response string, answer bool) {
	if !strings.HasPrefix(msg.Text, "??") {
		return "", false
	}
	reqText := strings.Replace(strings.TrimSpace(strings.TrimPrefix(msg.Text, "??")), " ", "+", -1)
	reqURL := fmt.Sprintf("https://duckduckgo-duckduckgo-zero-click-info.p.mashape.com/?format=json&no_html=1&no_redirect=1&q=%s&skip_disambig=1", reqText)

	client := http.Client{Timeout: time.Second * 5}

	req, err := makeHttpRequest(reqURL)
	if err != nil {
		log.Printf("[WARN] failed to make request %s, error=%v", reqURL, err)
		return "", false
	}
	req.Header.Set("X-Mashape-Key", d.mashapeKey)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[WARN] failed to send request %s, error=%v", reqURL, err)
		return "", false
	}
	defer resp.Body.Close()

	duckResp := struct {
		AbstractText   string `json:"AbstractText"`
		AbstractSource string `json:"AbstractSource"`
		AbstractURL    string `json:"AbstractURL"`
		Image          string `json:"Image"`
	}{}
	if err = json.NewDecoder(resp.Body).Decode(&duckResp); err != nil {
		log.Printf("[WARN] failed to convert from json, error=%v", err)
		return "", false
	}

	mdLink := func(inp string) string {
		result := strings.Replace(inp, "(", "%28", -1)
		result = strings.Replace(result, ")", "%29", -1)
		return result
	}

	if duckResp.AbstractText == "" {
		return fmt.Sprintf("_не в силах. но могу помочь_ [это поискать](https://duckduckgo.com/?q=%s)", mdLink(reqText)), true
	}

	respMD := fmt.Sprintf("- %s\n[%s](%s)", duckResp.AbstractText, duckResp.AbstractSource, mdLink(duckResp.AbstractURL))
	return respMD, true
}

// ReactOn keys
func (d Duck) ReactOn() []string {
	return []string{"??"}
}
