package openai

import (
	"fmt"
	"github.com/radio-t/super-bot/app/bot"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LimitedMessageHistory(t *testing.T) {
	tests := []struct {
		name  string
		limit int
	}{
		{name: "Limit 5", limit: 5},
		{name: "Limit 10", limit: 10},
		{name: "Limit 20", limit: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			history := NewLimitedMessageHistory(tt.limit)

			// Add `limit` messages to the storage
			for i := 0; i < tt.limit; i++ {
				history.Add(bot.Message{
					ID:   i,
					Text: fmt.Sprintf("test %d", i),
				})

				assert.Equal(t, i+1, len(history.messages))
			}

			assert.Equal(t, tt.limit, len(history.messages))
			for i := 0; i < tt.limit; i++ {
				assert.Equal(t, i, history.messages[i].ID)
				assert.Equal(t, fmt.Sprintf("test %d", i), history.messages[i].Text)
			}

			// Add messages to the storage. This should remove the oldest messages
			for j := 0; j < 3; j++ {
				newID := tt.limit + j
				history.Add(bot.Message{
					ID:   newID,
					Text: fmt.Sprintf("test %d", newID),
				})

				assert.Equal(t, tt.limit, len(history.messages))
				for i := 0; i < tt.limit; i++ {
					expectedID := i + j + 1
					assert.Equal(t, expectedID, history.messages[i].ID)
					assert.Equal(t, fmt.Sprintf("test %d", expectedID), history.messages[i].Text)
				}
			}

		})
	}

}
