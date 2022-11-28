package bot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWhenBot(t *testing.T) {
	t.Parallel()

	b := NewWhen()

	t.Run("react_on", func(t *testing.T) {
		assert.Equal(t, []string{"когда?", "when?"}, b.ReactOn())
	})

	t.Run("help", func(t *testing.T) {
		assert.Equal(t, "когда?, when? _– расписание эфиров Радио-Т_\n", b.Help())
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
			exp: "[каждую субботу, 20:00 UTC](https://radio-t.com/online/)\nНачнется через 18ч 59мин",
		},
		{
			in:  time.Date(2022, 1, 1, 20, 00, 0, 0, time.UTC),
			exp: "[каждую субботу, 20:00 UTC](https://radio-t.com/online/)\nНачнется через пару секунд",
		},
		{
			in:  time.Date(2022, 1, 1, 20, 01, 0, 0, time.UTC),
			exp: "[каждую субботу, 20:00 UTC](https://radio-t.com/online/)\nНачался 1мин назад. \nСкорее всего еще идет. \nСледующий через 6дн 23ч 59мин",
		},
		{
			in:  time.Date(2022, 1, 1, 22, 01, 0, 0, time.UTC),
			exp: "[каждую субботу, 20:00 UTC](https://radio-t.com/online/)\nНачнется через 6дн 21ч 59мин",
		},
	}

	for _, row := range table {
		t.Run("", func(t *testing.T) {
			res := when(row.in)
			assert.Equal(t, row.exp, res)
		})
	}
}

func TestWhenBot_closestPrevNextStreams(t *testing.T) {
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
			res1, res2 := closestPrevNextShows(row.in)
			assert.Equal(t, row.exp1, res1)
			assert.Equal(t, row.exp2, res2)
		})
	}
}
