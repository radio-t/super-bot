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
	Bots    bot.Interface
	GroupID string
	Debug   bool

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
			log.Printf("[INFO] receive update: %+v, %+v", update.Message, update)

			if update.Message == nil {
				log.Printf("[DEBUG] empty message body")
				continue
			}

			msg := l.convert(update.Message)
			l.Save(msg) // save to report

			log.Printf("[DEBUG] incoming msg: %+v", msg)

			// check for ban
			if b := l.check(msg.From); b.active {
				if b.new {
					m := fmt.Sprintf("@%s _тебя слишком много, отдохни..._", msg.From.Username)
					tbMsg := tbapi.NewMessage(update.Message.Chat.ID, m)
					tbMsg.ParseMode = tbapi.ModeMarkdown
					if res, e := l.botAPI.Send(tbMsg); e != nil {
						log.Printf("[WARN] failed to send, %v", e)
					} else {
						l.saveBotMessage(&res)
					}
				}
				continue
			}

			if resp, send := l.Bots.OnMessage(msg); send {
				log.Printf("[DEBUG] bot response - %+v", resp)
				tbMsg := tbapi.NewMessage(update.Message.Chat.ID, resp)
				tbMsg.ParseMode = tbapi.ModeMarkdown
				if res, e := l.botAPI.Send(tbMsg); err != nil {
					log.Printf("[WARN] can't send tbMsg to telegram, %v", e)
				} else {
					l.saveBotMessage(&res)
				}
			}

		case msg := <-l.msgs.ch: // publish messages from outside clients
			chat, e := l.botAPI.GetChat(tbapi.ChatConfig{SuperGroupUsername: l.GroupID})
			if err != nil {
				return errors.Wrapf(err, "can't get chat for %s", l.GroupID)
			}
			tbMsg := tbapi.NewMessage(chat.ID, msg)
			tbMsg.ParseMode = tbapi.ModeMarkdown
			if res, err := l.botAPI.Send(tbMsg); err != nil {
				log.Printf("[WARN] can't send msg to telegram, %v", e)
			} else {
				l.saveBotMessage(&res)
			}
		}
	}
}

// Submit message text to telegram's group
func (l *TelegramListener) Submit(ctx context.Context, text string) error {
	l.msgs.once.Do(func() { l.msgs.ch = make(chan string, 100) })

	select {
	case <-ctx.Done():
		return ctx.Err()
	case l.msgs.ch <- text:
	}
	return nil
}

func (l *TelegramListener) saveBotMessage(msg *tbapi.Message) {
	l.Save(l.convert(msg))
}

func (l *TelegramListener) convert(msg *tbapi.Message) bot.Message {
	message := bot.Message{
		From: bot.User{
			Username:    msg.From.UserName,
			DisplayName: msg.From.FirstName + " " + msg.From.LastName,
		},
		Sent: msg.Time(),
		Text: msg.Text,
	}

	if msg.Photo != nil && len(*msg.Photo) > 0 {
		message.Picture = &bot.Picture{
			Image: bot.Image{
				Source: bot.Source{
					FileID: (*msg.Photo)[0].FileID,
					Width:  (*msg.Photo)[0].Width,
					Height: (*msg.Photo)[0].Height,
				},
				Sources: l.convertPhotoSizes(*msg.Photo),
			},
			Caption: msg.Caption,
		}
	}

	if msg.Sticker != nil {
		message.Picture = &bot.Picture{
			Image: bot.Image{
				Source: bot.Source{
					FileID: msg.Sticker.Thumbnail.FileID + ".jpg",
					Width:  msg.Sticker.Thumbnail.Width,
					Height: msg.Sticker.Thumbnail.Height,
					Alt:    msg.Sticker.Emoji,
				},
			},
			Sources: l.convertSticker(*msg.Sticker),
		}
	}

	return message
}

func (l *TelegramListener) convertPhotoSizes(sizes []tbapi.PhotoSize) []bot.Source {
	var result []bot.Source

	for _, size := range sizes {
		result = append(
			result,
			bot.Source{
				FileID: size.FileID,
				Size:   size.FileSize,
				Width:  size.Width,
				Height: size.Height,
			},
		)
	}

	return result
}

func (l *TelegramListener) convertSticker(sticker tbapi.Sticker) []bot.Source {
	var result []bot.Source

	result = append(
		result,
		bot.Source{
			FileID: sticker.Thumbnail.FileID,
			Type:   "webp",
			Size:   sticker.Thumbnail.FileSize,
		},
	)

	result = append(
		result,
		bot.Source{
			FileID: sticker.Thumbnail.FileID + ".jpg",
			Type:   "jpeg",
		},
	)

	return result
}
