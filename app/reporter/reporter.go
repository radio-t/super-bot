package reporter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/radio-t/super-bot/app/bot"
	"net/http"
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
	httpCl    httpClient
	chatID    string
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
		chatID: chatID,
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
		case <-time.Tick(time.Second * 5): // flush on 5 seconds inactivity
			if err := writeBuff(); err != nil {
				log.Printf("[WARN] failed to write reporter buffer, %v", err)
			}
		}
	}
}

//go:generate moq -out mock_reporter.go . httpClient

type httpClient interface {
	Get(url string) (*http.Response, error)
}

func (l *Reporter) messageExists(msgID int) bool {
	resp, err := l.httpCl.Get(fmt.Sprintf("https://t.me/%s/%d?single", l.chatID, msgID))
	if err != nil {
		log.Printf("[WARN] failed to check message existence, %v", err)
		return false
	}
	defer resp.Body.Close() // nolint

	return resp.StatusCode == http.StatusFound
}
