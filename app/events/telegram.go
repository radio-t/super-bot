package events

import (
	"context"
	"fmt"
	"sync"

	log "github.com/go-pkgz/lgr"
	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"

	"github.com/radio-t/gitter-rt-bot/app/bot"
	"github.com/radio-t/gitter-rt-bot/app/reporter"
)

// TelegramListener listens to tg update, forward to bots and send back responses
type TelegramListener struct {
	Terminator
	Token string
	reporter.Reporter
	Bots        bot.Interface
	ChannelID   string
	BotUserName string
	Debug       bool

	botAPI *tbapi.BotAPI

	msgs struct {
		once sync.Once
		ch   chan string
	}
}

// Do process all events, blocked call
func (l *TelegramListener) Do(ctx context.Context) (err error) {
	if l.botAPI, err = tbapi.NewBotAPI(l.Token); err != nil {
		return errors.Wrap(err, "can't make telegram bot")
	}
	l.botAPI.Debug = l.Debug
	l.msgs.once.Do(func() { l.msgs.ch = make(chan string, 100) })

	u := tbapi.NewUpdate(0)
	u.Timeout = 60

	var updates tbapi.UpdatesChannel
	if updates, err = l.botAPI.GetUpdatesChan(u); err != nil {
		return errors.Wrap(err, "can't get updates channel")
	}

	for {
		select {

		case <-ctx.Done():
			return ctx.Err()

		case update, ok := <-updates:
			if !ok {
				return errors.Errorf("telegram update chan closed")
			}
			log.Printf("%+v %+v", update.ChannelPost, update)
			msg := bot.Message{
				Text: update.ChannelPost.Text,
				From: bot.User{
					// ID:          strconv.Itoa(update.ChannelPost.Chat.ID),
					Username:    update.ChannelPost.Chat.UserName,
					DisplayName: update.ChannelPost.Chat.FirstName + " " + update.ChannelPost.Chat.LastName,
				},
				Sent: update.ChannelPost.Time(),
			}

			l.Save(msg) // save to report

			if msg.From.Username == l.BotUserName {
				log.Printf("[DEBUG] ignore %+v", msg.Text)
				continue
			}

			// check for ban
			if ban := l.check(msg.From); ban.active {
				if ban.new {
					m := fmt.Sprintf("@%s _тебя слишком много, отдохни ..._", msg.From.Username)
					tbMsg := tbapi.NewMessage(update.ChannelPost.Chat.ID, m)
					tbMsg.ParseMode = "markdown"
					if _, err := l.botAPI.Send(tbMsg); err != nil {
						log.Printf("[WARN] failed to send, %v", err)
					}
				}
				continue
			}

			if resp, send := l.Bots.OnMessage(msg); send {
				log.Printf("[DEBUG] bot response - %+v", resp)
				tbMsg := tbapi.NewMessage(update.ChannelPost.Chat.ID, resp)
				tbMsg.ParseMode = "markdown"
				if _, err := l.botAPI.Send(tbMsg); err != nil {
					log.Printf("[WARN] can't send tbMsg to telegram, %v", err)
				}
			}

		case msg := <-l.msgs.ch: // publish messages from outside clients
			chat, err := l.botAPI.GetChat(tbapi.ChatConfig{SuperGroupUsername: l.ChannelID})
			if err != nil {
				return errors.Wrapf(err, "can't get chat for %s", l.ChannelID)
			}
			tbMsg := tbapi.NewMessage(chat.ID, msg)
			tbMsg.ParseMode = "markdown"
			if _, err := l.botAPI.Send(tbMsg); err != nil {
				log.Printf("[WARN] can't send msg to telegram, %v", err)
			}
		}
	}
}

// Submit message text to telegram's channel
func (l *TelegramListener) Submit(ctx context.Context, text string) error {
	l.msgs.once.Do(func() { l.msgs.ch = make(chan string, 100) })

	select {
	case <-ctx.Done():
		return ctx.Err()
	case l.msgs.ch <- text:
	}
	return nil
}
