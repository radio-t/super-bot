package bot

import (
	"fmt"
	"time"
)

// Day is one day duration
const Day = 24 * time.Hour

// HumanizeDuration converts time.Duration to readable format
func HumanizeDuration(d time.Duration) string {
	seconds := int64(d.Seconds()) % 60
	minutes := int64(d.Minutes()) % 60
	hours := int64(d.Hours()) % 24
	days := int64(d.Hours()) / 24

	result := ""
	first := true
	if days > 0 {
		if !first {
			result += " "
		}
		result += fmt.Sprintf("%vдн", days)
		first = false
	}
	if hours > 0 {
		if !first {
			result += " "
		}
		result += fmt.Sprintf("%vч", hours)
		first = false
	}
	if minutes > 0 {
		if !first {
			result += " "
		}
		result += fmt.Sprintf("%vмин", minutes)
		first = false
	}
	if seconds > 0 {
		if !first {
			result += " "
		}
		result += fmt.Sprintf("%vсек", seconds)
		first = false
	}

	return result
}
