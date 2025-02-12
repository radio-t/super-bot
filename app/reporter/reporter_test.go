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
	"path"
)

var msg = bot.Message{ID: 101, Text: "1st"}

func TestNewLogger(t *testing.T) {
	t.Run("logger saves messages", func(t *testing.T) {
		tbl := []struct {
			count   int
			timeout time.Duration
		}{
			{101, 100 * time.Millisecond},
			{1, 6 * time.Second},
		}

		for i, tt := range tbl {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				p := path.Join(os.TempDir(), "superbot_logs", strconv.Itoa(i))
				defer os.RemoveAll(p)

				clientMock := &httpClientMock{
					GetFunc: func(url string) (*http.Response, error) {
						assert.Equal(t, "https://t.me/radio_t_chat/101?single", url)
						return &http.Response{
							StatusCode: 302,
							Body:       io.NopCloser(bytes.NewBuffer([]byte(""))),
						}, nil
					},
				}

				reporter := NewLogger(p, 0, "radio_t_chat")
				reporter.httpCl = clientMock
				assert.NotNil(t, reporter)
				assert.DirExists(t, p)

				for i = 0; i < tt.count; i++ {
					reporter.Save(&msg)
				}
				time.Sleep(tt.timeout)
				logfile := fmt.Sprintf("%s/%s.log", p, time.Now().Format("20060102"))
				assert.FileExists(t, logfile)
				require.NoError(t, os.Remove(logfile))
			})
		}
	})

	t.Run("logger skips deleted messages", func(t *testing.T) {
		path, err := os.MkdirTemp("", "superbot_logs")
		require.NoError(t, err)
		defer os.RemoveAll(path)

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
		reporter := NewLogger(path, 500*time.Millisecond, "radio_t_chat")
		reporter.httpCl = clientMock
		assert.NotNil(t, reporter)
		assert.DirExists(t, path)

		startedAt = time.Now()
		reporter.Save(&bot.Message{ID: 101, Text: "something"})

		time.Sleep(6 * time.Second) // wait for forced flush
		logfile := fmt.Sprintf("%s/%s.log", path, time.Now().Format("20060102"))
		assert.FileExists(t, logfile)
		require.NoError(t, os.Remove(logfile))
	})
}
