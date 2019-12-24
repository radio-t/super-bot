package events

import (
	"context"

	log "github.com/go-pkgz/lgr"
	tb "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"

	"github.com/radio-t/gitter-rt-bot/app/bot"
	"github.com/radio-t/gitter-rt-bot/app/reporter"
)

type TelegramListener struct {
	Terminator
	Token string
	reporter.Reporter
	Bots        bot.Interface
	ChannelID   string
	BotUserName string
}

// Do process all events, blocked call
func (l *TelegramListener) Do(ctx context.Context) error {
	b, err := tb.NewBotAPI(l.Token)
	if err != nil {
		return errors.Wrap(err, "can't make telegram bot")
	}
	b.Debug = true

	u := tb.NewUpdate(0)
	u.Timeout = 60
	updates, err := b.GetUpdatesChan(u)
	if err != nil {
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

			l.Save(msg)

			if msg.From.Username == l.BotUserName {
				log.Printf("[DEBUG] ignore %+v", msg.Text)
				continue
			}

			if resp, send := l.Bots.OnMessage(msg); send {
				log.Printf("[DEBUG] bot response - %+v", resp)
				msg := tb.NewMessage(update.ChannelPost.Chat.ID, resp)
				// msg.ReplyToMessageID = update.ChannelPost.MessageID
				msg.ParseMode = "markdown"
				if _, err := b.Send(msg); err != nil {
					log.Printf("[WARN] can't send msg to telegram, %v", err)
				}
			}

		}
	}
}
