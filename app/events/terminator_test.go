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
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))

	// banned
	assert.Equal(t, ban{active: true, new: true}, term.check(bot.User{Username: "user"}, time.Now()))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: true, new: false}, term.check(bot.User{Username: "user"}, time.Now()))

	// ban expired
	time.Sleep(510 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))

	// trigger ban again
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: true, new: true}, term.check(bot.User{Username: "user"}, time.Now()))
}

func TestTerminator_checkAdmin(t *testing.T) {
	term := Terminator{
		BanDuration:   500 * time.Millisecond,
		BanPenalty:    2,
		AllowedPeriod: 100 * time.Millisecond,
		Exclude:       []string{"umputun"},
	}

	// try to trigger ban
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "umputun"}, time.Now()))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "umputun"}, time.Now()))
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "umputun"}, time.Now()))
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "umputun"}, time.Now()))
}

func TestTerminator_checkOk(t *testing.T) {
	term := Terminator{
		BanDuration:   500 * time.Millisecond,
		BanPenalty:    3,
		AllowedPeriod: 50 * time.Millisecond,
		Exclude:       []string{"umputun"},
	}

	// trigger ban
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
	time.Sleep(51 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
	time.Sleep(51 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
	time.Sleep(51 * time.Millisecond)
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now()))
}

func TestTerminator_ignoreOldMessages(t *testing.T) {
	term := Terminator{
		BanDuration:   500 * time.Millisecond,
		BanPenalty:    2,
		AllowedPeriod: 10 * time.Millisecond,
	}

	// ignore old messages
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now().Add(-12*time.Millisecond)))
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now().Add(-11*time.Millisecond)))
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now().Add(-10*time.Millisecond)))

	// handle messages that entered "allowed period" window
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now().Add(-9*time.Millisecond))) // penalty = 0
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now().Add(-8*time.Millisecond))) // penalty = 1
	assert.Equal(t, ban{active: false, new: false}, term.check(bot.User{Username: "user"}, time.Now().Add(-7*time.Millisecond))) // penalty = 2
	assert.Equal(t, ban{active: true, new: true}, term.check(bot.User{Username: "user"}, time.Now().Add(-6*time.Millisecond)))   // ban
}
