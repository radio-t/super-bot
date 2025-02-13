package reporter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/radio-t/super-bot/app/bot"
	"net/http"
	"github.com/go-pkgz/repeater"
	"context"
)

type msgEntry struct {
	MessageID int
	Data      string
}

// Reporter collects all messages and saves to plain file
type Reporter struct {
	logsPath  string
	messages  chan msgEntry
	saveDelay time.Duration
	chatID    string
	httpCl    httpClient
	repeater  *repeater.Repeater
}

//go:generate moq -out mock_reporter.go . httpClient

type httpClient interface {
	Get(url string) (*http.Response, error)
}

// NewLogger makes new reporter bot
func NewLogger(logs string, delay time.Duration, chatID string) (result *Reporter) {
	log.Printf("[INFO] new reporter, path=%s", logs)
	if err := os.MkdirAll(logs, 0o750); err != nil {
		log.Printf("[WARN] can't make logs dir %s, %v", logs, err)
	}
	result = &Reporter{
		logsPath:  logs,
		messages:  make(chan msgEntry, 1000),
		saveDelay: delay,
		httpCl: &http.Client{
			Timeout: time.Second * 5,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				// don't follow redirects, we need to check status code
				return http.ErrUseLastResponse
			},
		},
		repeater: repeater.NewDefault(3, 2*time.Second),
		chatID:   chatID,
	}
	go result.activate()
	return result
}

// Save to log channel, non-blocking and skip if needed
func (l *Reporter) Save(msg *bot.Message) {
	if msg.Text == "" && msg.Image == nil {
		log.Printf("[DEBUG] message not saved to log: no text or image = irrelevant, msg id: %d", msg.ID)
		return
	}

	bdata, err := json.Marshal(&msg)
	if err != nil {
		log.Printf("[WARN] failed to log, error %v", err)
		return
	}

	select {
	case l.messages <- msgEntry{MessageID: msg.ID, Data: string(bdata) + "\n"}:
	default:
		log.Printf("[WARN] can't buffer log entry %v", msg)
	}
}

func (l *Reporter) activate() {
	log.Print("[INFO] activate reporter")
	buffer := make([]string, 0, 100)
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	writeBuff := func() error {
		if len(buffer) == 0 {
			return nil
		}
		// nolint
		fh, err := os.OpenFile(fmt.Sprintf("%s/%s.log", l.logsPath, time.Now().Format("20060102")),
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0660)

		if err != nil {
			log.Printf("[WARN] failed to log, %v", err)
			return err
		}
		defer fh.Close() // nolint
		for _, rec := range buffer {
			if _, err = fh.WriteString(rec); err != nil {
				log.Printf("[WARN] failed to write log, %v", err)
			}
		}

		log.Printf("[DEBUG] wrote %d log entries", len(buffer))
		buffer = buffer[:0]
		return nil
	}

	for {
		select {
		case entry := <-l.messages:
			// don't save right away, wait for antispam checks
			time.Sleep(l.saveDelay)

			if !l.messageExists(entry.MessageID) {
				log.Printf("[DEBUG] message %d has been deleted, skipping", entry.MessageID)
				continue
			}

			buffer = append(buffer, entry.Data)
			if len(buffer) >= 100 { // forced flush every 100 records
				if err := writeBuff(); err != nil {
					log.Printf("[WARN] failed to write reporter buffer, %v", err)
				}
			}
		case <-ticker.C: // flush on 5 seconds inactivity
			if err := writeBuff(); err != nil {
				log.Printf("[WARN] failed to write reporter buffer, %v", err)
			}
		}
	}
}

// messageExists checks if message wasn't deleted by a user, to prevent
// spam being saved to logs.
// It uses a hacky way to check if message exists, by
// requesting a link to the message with "single" query parameter.
// ref: https://core.telegram.org/api/links#message-links
// telegram returns 302 redirect to the same page without query, if it exists
// and 200 with the "download telegram" webpage, if it isn't
func (l *Reporter) messageExists(msgID int) bool {
	var resp *http.Response
	var err error

	fn := func() error {
		//nolint:bodyclose // for some reason it gives false positives
		resp, err = l.httpCl.Get(fmt.Sprintf("https://t.me/%s/%d?single", l.chatID, msgID))
		if err != nil {
			return fmt.Errorf("get: %w", err)
		}
		defer resp.Body.Close() //nolint

		if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusOK {
			return nil
		}

		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err = l.repeater.Do(context.TODO(), fn); err != nil {
		log.Printf("[WARN] failed to check message existence, %v", err)
		return false
	}

	return resp.StatusCode == http.StatusFound
}
