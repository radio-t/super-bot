package bot

import (
	"fmt"
	"github.com/dustin/go-humanize"
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

	var whenCountdown string
	if diffToPrev < avgDuration {
		whenCountdown = fmt.Sprintf(
			"\nНачался %s назад. \nСкорее всего еще идет. \nСледующий через %s",
			humanizeDuration(now, prevStream),
			humanizeDuration(now, nextStream),
		)
	} else {
		whenCountdown = fmt.Sprintf(
			"\nНачнется через %s",
			humanizeDuration(now, nextStream),
		)
	}

	return whenPrefix + whenCountdown
}

// NOTE: copied from humanize.defaultMagnitudes
var ruMagnitudes = []humanize.RelTimeMagnitude{
	{time.Second, "пару секунд", time.Second},
	{2 * time.Second, "1s", 1},
	{time.Minute, "%ds", time.Second},
	{2 * time.Minute, "1m", 1},
	{time.Hour, "%dm", time.Minute},
	{2 * time.Hour, "1h", 1},
	{3 * time.Hour, "%dh", time.Hour},
	{humanize.Day, "%dh", time.Hour},
	{2 * humanize.Day, "1d", 1},
	{5 * humanize.Day, "%dd", humanize.Day},
	{humanize.Week, "%dd", humanize.Day},
	{2 * humanize.Week, "1w", 1},
	{humanize.Month, "%dw", humanize.Week},
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

func humanizeDuration(baseDt, nextDt time.Time) string {
	return humanize.CustomRelTime(baseDt, nextDt, "", "", ruMagnitudes)
}
