package reporter

import (
	"testing"
	"github.com/radio-t/super-bot/app/bot"
	"os"
	"net/http"
	"time"
	"github.com/stretchr/testify/assert"
	"strconv"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"bytes"
)

var logs = "logs"
var msg = bot.Message{ID: 101, Text: "1st"}

func TestNewLogger(t *testing.T) {
	t.Run("logger saves messages", func(t *testing.T) {
		defer os.RemoveAll(logs)
		clientMock := &httpClientMock{
			GetFunc: func(url string) (*http.Response, error) {
				assert.Equal(t, "https://t.me/radio_t_chat/101?single", url)
				return &http.Response{
					StatusCode: 302,
					Body:       io.NopCloser(bytes.NewBuffer([]byte(""))),
				}, nil
			},
		}

		reporter := NewLogger(logs, 500*time.Millisecond, "radio_t_chat", clientMock)
		assert.NotNil(t, reporter)
		assert.DirExists(t, logs)

		tbl := []struct {
			count   int
			timeout time.Duration
		}{
			{101, 100 * time.Millisecond},
			{1, 6 * time.Second},
		}

		for i, tt := range tbl {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				for i = 0; i < tt.count; i++ {
					reporter.Save(&msg)
				}
				time.Sleep(tt.timeout)
				logfile := fmt.Sprintf("%s/%s.log", logs, time.Now().Format("20060102"))
				assert.FileExists(t, logfile)
				err := os.Remove(logfile)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("logger skips deleted messages", func(t *testing.T) {
		var startedAt time.Time
		clientMock := &httpClientMock{
			GetFunc: func(url string) (*http.Response, error) {
				actualDelay := time.Since(startedAt)
				assert.Equal(t, "https://t.me/radio_t_chat/101?single", url)
				assert.True(t, actualDelay > 500*time.Millisecond && actualDelay < 600*time.Millisecond,
					"delay expected 500ms, got %v", actualDelay)
				return &http.Response{
					StatusCode: 302,
					Body:       io.NopCloser(bytes.NewBuffer([]byte(""))),
				}, nil
			},
		}
		reporter := NewLogger(logs, 500*time.Millisecond, "radio_t_chat", clientMock)
		assert.NotNil(t, reporter)
		assert.DirExists(t, logs)

		startedAt = time.Now()
		reporter.Save(&bot.Message{ID: 101, Text: "something"})

		time.Sleep(6 * time.Second) // wait for forced flush
		logfile := fmt.Sprintf("%s/%s.log", logs, time.Now().Format("20060102"))
		assert.FileExists(t, logfile)
		require.NoError(t, os.Remove(logfile))
	})
}
