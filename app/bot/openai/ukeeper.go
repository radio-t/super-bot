package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// UKeeperClient is a client for uKeeper API
type UKeeperClient struct {
	*http.Client
	API   string
	Token string
}

// Get gets title and content for the link with uKeeper API.
// Important: this method returns error if content type is not text/*
func (u UKeeperClient) Get(link string) (title, content string, err error) {
	rl := fmt.Sprintf("%s?token=%s&url=%s", u.API, u.Token, link)
	resp, err := u.Client.Get(rl)
	if err != nil {
		return "", "", fmt.Errorf("can't get summary for %s: %w", link, err)
	}
	defer resp.Body.Close() // nolint
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("can't get summary for %s: %d", link, resp.StatusCode)
	}

	urResp := struct {
		Title   string `json:"Title"`
		Content string `json:"Content"`
		Type    string `json:"type"`
	}{}
	if decErr := json.NewDecoder(resp.Body).Decode(&urResp); decErr != nil {
		return "", "", fmt.Errorf("can't decode summary for %s: %w", link, decErr)
	}

	// if content type is not text, we can't summarize it
	if !strings.Contains(urResp.Type, "text") {
		return "", "", fmt.Errorf("bad content type %s: %s", link, urResp.Type)
	}

	return urResp.Title, urResp.Content, nil
}
