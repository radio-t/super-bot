package reporter

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/radio-t/gitter-rt-bot/app/bot"
)

var logs = "logs"
var msg = bot.Message{Text: "1st"}

func TestNewLogger(t *testing.T) {
	defer os.RemoveAll(logs)
	reporter := NewLogger(logs)
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
				reporter.Save(msg)
			}
			time.Sleep(tt.timeout)
			logfile := fmt.Sprintf("%s/%s.log", logs, time.Now().Format("20060102"))
			assert.FileExists(t, logfile)
			err := os.Remove(logfile)
			assert.NoError(t, err)
		})
	}
}
