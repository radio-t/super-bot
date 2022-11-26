package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWhenBot(t *testing.T) {
	t.Parallel()

	now := time.Date(2006, 8, 27, 11, 33, 0, 0, time.UTC)
	b := NewWhen(
		[]string{"test1?", "test2?"},
		func() time.Time { return now },
	)

	t.Run("react_on", func(t *testing.T) {
		assert.Equal(t, []string{"test1?", "test2?"}, b.ReactOn())
	})

	t.Run("help", func(t *testing.T) {
		assert.Equal(t, "test1?, test2? _– расписание эфиров Радио-Т_\n", b.Help())
	})
}

func TestWhenBot_when(t *testing.T) {
	t.Parallel()

	table := []struct {
		in  time.Time
		exp string
	}{
		{
			in:  time.Date(2022, 1, 1, 1, 1, 0, 0, time.UTC),
			exp: "[каждую субботу, 20:00 UTC](https://radio-t.com/online/)\nНачнется через 18h 59m",
		},
		{
			in:  time.Date(2022, 1, 1, 20, 00, 0, 0, time.UTC),
			exp: "[каждую субботу, 20:00 UTC](https://radio-t.com/online/)\nНачнется через пару секунд",
		},
		{
			in:  time.Date(2022, 1, 1, 20, 01, 0, 0, time.UTC),
			exp: "[каждую субботу, 20:00 UTC](https://radio-t.com/online/)\nНачался 1m назад. \nСкорее всего еще идет. \nСледующий через 6d 23h",
		},
		{
			in:  time.Date(2022, 1, 1, 22, 01, 0, 0, time.UTC),
			exp: "[каждую субботу, 20:00 UTC](https://radio-t.com/online/)\nНачнется через 6d 21h",
		},
	}

	for _, row := range table {
		t.Run("", func(t *testing.T) {
			res := when(row.in)
			assert.Equal(t, row.exp, res)
		})
	}

}

func TestWhenBot_humanizeDuration(t *testing.T) {
	t.Parallel()

	table := []struct {
		in         time.Duration
		defaultVal string
		exp        string
	}{
		{
			in:  0,
			exp: "",
		},
		{
			in:  11 * time.Second,
			exp: "11s",
		},
		{
			in:  1 * time.Minute,
			exp: "1m",
		},
		{
			in:  3*time.Minute + 59*time.Second,
			exp: "3m 59s",
		},
		{
			in:  2*time.Hour + 13*time.Second,
			exp: "2h 13s",
		},
		{
			in:  3*24*time.Hour + 4*time.Hour + 5*time.Minute + 6*time.Second,
			exp: "3d 4h",
		},
		{
			in:         0,
			defaultVal: "right now",
			exp:        "right now",
		},
	}

	for _, row := range table {
		t.Run("", func(t *testing.T) {
			res := humanizeDuration(row.in, row.defaultVal)
			assert.Equal(t, row.exp, res)
		})
	}
}

func TestWhenBot_closestPrevNextWeekdays(t *testing.T) {
	t.Parallel()

	table := []struct {
		in         time.Time
		exp1, exp2 time.Time
	}{
		{
			in:   time.Date(2022, 11, 21, 11, 33, 0, 0, time.UTC),
			exp1: time.Date(2022, 11, 19, 20, 0, 0, 0, time.UTC),
			exp2: time.Date(2022, 11, 26, 20, 0, 0, 0, time.UTC),
		},
		{
			in:   time.Date(2022, 11, 26, 19, 33, 0, 0, time.UTC),
			exp1: time.Date(2022, 11, 19, 20, 0, 0, 0, time.UTC),
			exp2: time.Date(2022, 11, 26, 20, 0, 0, 0, time.UTC),
		},
		{
			in:   time.Date(2022, 11, 21, 11, 33, 0, 0, time.UTC),
			exp1: time.Date(2022, 11, 19, 20, 0, 0, 0, time.UTC),
			exp2: time.Date(2022, 11, 26, 20, 0, 0, 0, time.UTC),
		},
		{
			in:   time.Date(2006, 8, 27, 19, 33, 0, 0, time.UTC), // вс
			exp1: time.Date(2006, 8, 26, 20, 0, 0, 0, time.UTC),
			exp2: time.Date(2006, 9, 2, 20, 0, 0, 0, time.UTC),
		},
		{
			in:   time.Date(2022, 11, 26, 20, 33, 0, 0, time.UTC), // вс
			exp1: time.Date(2022, 11, 26, 20, 0, 0, 0, time.UTC),
			exp2: time.Date(2022, 12, 3, 20, 0, 0, 0, time.UTC),
		},
	}

	for _, row := range table {
		t.Run("", func(t *testing.T) {
			res1, res2 := closestPrevNextWeekdays(row.in, time.Saturday, 20, 0)
			assert.Equal(t, row.exp1, res1)
			assert.Equal(t, row.exp2, res2)
		})
	}
}
