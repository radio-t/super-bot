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
	return GenHelpMsg(w.ReactOn(), "расписание эфиров Радио-Т")
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
	prevStream, nextStream := closestPrevNextShows(now)
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

// closestPrevNextShows returns closest next `weekday` at `hour`:`minute` after `t`.
func closestPrevNextShows(t time.Time) (prev, next time.Time) {
	const week = 7 * Day

	next = t.AddDate(0, 0, int(time.Saturday-t.Weekday()))
	next = time.Date(next.Year(), next.Month(), next.Day(), 20, 0, 0, 0, time.UTC)

	if t.After(next) {
		return next, next.Add(week)
	}

	return next.Add(-week), next
}
