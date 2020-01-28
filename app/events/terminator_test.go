package events

import (
	"testing"
	"time"

	"github.com/radio-t/super-bot/app/bot"
	"github.com/stretchr/testify/assert"
)

func TestTerminator_checkTerminate(t *testing.T) {
	term := Terminator{
		BanDuration:   500 * time.Millisecond,
		BanPenalty:    2,
		AllowedPeriod: 100 * time.Millisecond,
		Exclude:       []string{"umputun"},
	}

	// trigger ban
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))

	// banned
	assert.Equal(t, ban{active: true, new: true}, term.check(bot.User{Username: "user"}))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: true, new: false}, term.check(bot.User{Username: "user"}))

	// ban expired
	time.Sleep(510 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))

	// trigger ban again
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: true, new: true}, term.check(bot.User{Username: "user"}))
}

func TestTerminator_checkAdmin(t *testing.T) {
	term := Terminator{
		BanDuration:   500 * time.Millisecond,
		BanPenalty:    2,
		AllowedPeriod: 100 * time.Millisecond,
		Exclude:       []string{"umputun"},
	}

	// try to trigger ban
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "umputun"}))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "umputun"}))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "umputun"}))
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "umputun"}))
}

func TestTerminator_checkOk(t *testing.T) {
	term := Terminator{
		BanDuration:   500 * time.Millisecond,
		BanPenalty:    3,
		AllowedPeriod: 50 * time.Millisecond,
		Exclude:       []string{"umputun"},
	}

	// trigger ban
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
	time.Sleep(51 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
	time.Sleep(51 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
	time.Sleep(51 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}))
}
