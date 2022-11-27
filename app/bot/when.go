package bot

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// When bot is answer on question "when the stream is started".
type When struct{}

// NewWhen makes a new When bot.
func NewWhen() *When {
	log.Printf("[INFO] new when bot is started")

	return &When{}
}

// Help returns help message
func (w *When) Help() string {
	return genHelpMsg(w.ReactOn(), "расписание эфиров Радио-Т")
}

// OnMessage returns one entry
func (w *When) OnMessage(msg Message) Response {
	if !contains(w.ReactOn(), msg.Text) {
		return Response{}
	}

	return Response{
		Text: when(time.Now()),
		Send: true,
	}
}

// ReactOn keys
func (w *When) ReactOn() []string {
	return []string{"когда?", "when?"}
}

func when(now time.Time) string {
	const avgDuration = 2 * time.Hour
	const whenPrefix = "[каждую субботу, 20:00 UTC](https://radio-t.com/online/)"
	const tooSmallDuration = "пару секунд"

	now = now.UTC()
	prevStream, nextStream := closestPrevNextWeekdays(now, time.Saturday, 20, 0)
	diffToPrev := -prevStream.Sub(now)
	diffToNext := nextStream.Sub(now)

	var whenCountdown string
	if diffToPrev < avgDuration {
		whenCountdown = fmt.Sprintf(
			"\nНачался %s назад. \nСкорее всего еще идет. \nСледующий через %s",
			humanizeDuration(diffToPrev, tooSmallDuration),
			humanizeDuration(diffToNext, tooSmallDuration),
		)
	} else {
		whenCountdown = fmt.Sprintf(
			"\nНачнется через %s",
			humanizeDuration(diffToNext, tooSmallDuration),
		)
	}

	return whenPrefix + whenCountdown
}

// closestPrevNextWeekdays returns closest next `weekday` at `hour`:`minute` after `t`.
func closestPrevNextWeekdays(t time.Time, weekday time.Weekday, hour, minute int) (time.Time, time.Time) {
	const daysInWeek = 7
	const week = daysInWeek * 24 * time.Hour

	daysDiff := int((daysInWeek + (weekday - t.Weekday())) % daysInWeek)
	year, month, day := t.AddDate(0, 0, daysDiff).Date()

	nextDt := time.Date(year, month, day, hour, minute, 0, 0, t.Location())
	if t.After(nextDt) {
		return nextDt, nextDt.Add(week)
	}

	return nextDt.Add(-week), nextDt
}

func humanizeDuration(d time.Duration, defaultVal string) string {
	const maxParts = 2
	units := []struct {
		duration time.Duration
		name     string
	}{
		{
			duration: 24 * time.Hour,
			name:     "d",
		},
		{
			duration: time.Hour,
			name:     "h",
		},
		{
			duration: time.Minute,
			name:     "m",
		},
		{
			duration: time.Second,
			name:     "s",
		},
	}

	var res []string
	for _, unit := range units {
		unitCount := int(d / unit.duration)
		if unitCount > 0 {
			res = append(res, fmt.Sprintf("%d%s", unitCount, unit.name))
			if len(res) == maxParts {
				break
			}

			d = d - time.Duration(unitCount)*unit.duration
		}
	}

	if len(res) == 0 {
		return defaultVal
	}

	return strings.Join(res, " ")
}
