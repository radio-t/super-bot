package bot

import (
	"fmt"
	"log"
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

	now = now.UTC()
	prevStream, nextStream := closestPrevNextStreams(now)
	diffToPrev := -prevStream.Sub(now)
	diffToNext := nextStream.Sub(now)

	var whenCountdown string
	if diffToPrev < avgDuration {
		whenCountdown = fmt.Sprintf(
			"\nНачался %s назад. \nСкорее всего еще идет. \nСледующий через %s",
			HumanizeDuration(diffToPrev),
			HumanizeDuration(diffToNext),
		)
	} else {
		whenCountdown = fmt.Sprintf(
			"\nНачнется через %s",
			HumanizeDuration(diffToNext),
		)
	}

	return whenPrefix + whenCountdown
}

// closestPrevNextStreams returns closest next `weekday` at `hour`:`minute` after `t`.
func closestPrevNextStreams(t time.Time) (time.Time, time.Time) {
	const streamStartWeekday = time.Saturday
	const streamStartHour = 20
	const streamStartMinute = 0
	const daysInWeek = 7
	const week = daysInWeek * 24 * time.Hour

	daysDiff := int((daysInWeek + (streamStartWeekday - t.Weekday())) % daysInWeek)
	year, month, day := t.AddDate(0, 0, daysDiff).Date()

	nextDt := time.Date(year, month, day, streamStartHour, streamStartMinute, 0, 0, t.Location())
	if t.After(nextDt) {
		return nextDt, nextDt.Add(week)
	}

	return nextDt.Add(-week), nextDt
}
