package events

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"
)

//go:generate moq --out mocks/submitter.go --pkg mocks --skip-ensure . submitter:Submitter
//go:generate moq --out mocks/openai_summary.go --pkg mocks --skip-ensure . openAISummary:OpenAISummary

// pinned defines translation map for messages pinned by bot
var pinned = map[string]string{
	"⚠️ Официальный кат! - https://stream.radio-t.com/": "⚠️ Вещание подкаста началось - https://stream.radio-t.com/",
}

// Rtjc is a listener for incoming rtjc commands. Publishes whatever it got from the socket
// compatible with the legacy rtjc bot. Primarily use case is to push news events from news.radio-t.com
type Rtjc struct {
	Port          int
	Submitter     submitter
	OpenAISummary openAISummary

	UrAPI    string
	UrToken  string
	URClient *http.Client
}

// submitter defines interface to submit (usually asynchronously) to the chat
type submitter interface {
	Submit(ctx context.Context, text string, pin bool) error
}

type openAISummary interface {
	Summary(text string) (response string, err error)
}

// Listen on Port accept and forward to telegram
func (l Rtjc) Listen(ctx context.Context) {
	log.Printf("[INFO] rtjc listener on port %d", l.Port)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", l.Port))
	if err != nil {
		log.Fatalf("[ERROR] can't listen on %d, %v", l.Port, err)
	}

	sendSummary := func(msg string) {
		if !strings.HasPrefix(msg, "⚠") {
			return
		}
		title, txt, err := l.summary(msg)
		if err != nil {
			log.Printf("[WARN] can't get summary, %v", err)
			return
		}
		if txt == "" {
			log.Printf("[WARN] empty summary for %q", msg)
			return
		}
		if serr := l.Submitter.Submit(ctx, title+"\n\n"+txt, false); serr != nil {
			log.Printf("[WARN] can't send summary, %v", serr)
		}
	}

	for {
		conn, e := ln.Accept()
		if e != nil {
			log.Printf("[WARN] can't accept, %v", e)
			time.Sleep(time.Second * 1)
			continue
		}
		if message, rerr := bufio.NewReader(conn).ReadString('\n'); rerr == nil {
			pin, msg := l.isPinned(message)
			if serr := l.Submitter.Submit(ctx, msg, pin); serr != nil {
				log.Printf("[WARN] can't send message, %v", serr)
			}
			sendSummary(msg)
		} else {
			log.Printf("[WARN] can't read message, %v", rerr)
		}
		_ = conn.Close()
	}
}

func (l Rtjc) isPinned(msg string) (ok bool, m string) {
	cleanedMsg := strings.TrimSpace(msg)
	cleanedMsg = strings.TrimSuffix(cleanedMsg, "\n")

	for k, v := range pinned {
		if strings.EqualFold(cleanedMsg, k) {
			resMsg := v
			if strings.TrimSpace(resMsg) == "" {
				resMsg = msg
			}
			return true, resMsg
		}
	}
	return false, msg
}

// summary returns short summary of the selected news article
func (l Rtjc) summary(msg string) (title, content string, err error) {
	re := regexp.MustCompile(`https?://[^\s"'<>]+`)
	link := re.FindString(msg)
	if strings.Contains(link, "radio-t.com") {
		return "", "", nil // ignore radio-t.com links
	}
	log.Printf("[DEBUG] summary for link:%s", link)

	rl := fmt.Sprintf("%s?token=%s&url=%s", l.UrAPI, l.UrToken, link)
	resp, err := l.URClient.Get(rl)
	if err != nil {
		return "", "", fmt.Errorf("can't get summary for %s: %w", link, err)
	}
	defer resp.Body.Close() // nolint
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("can't get summary for %s: %d", link, resp.StatusCode)
	}

	urResp := struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}{}
	if decErr := json.NewDecoder(resp.Body).Decode(&urResp); decErr != nil {
		return "", "", fmt.Errorf("can't decode summary for %s: %w", link, decErr)
	}

	res, err := l.OpenAISummary.Summary(urResp.Title + " - " + urResp.Content)
	if err != nil {
		return "", "", fmt.Errorf("can't get summary for %s: %w", link, err)
	}

	return urResp.Title, res, nil
}
