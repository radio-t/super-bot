package events

import (
	"context"
	"encoding/json"
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

			log.Printf("[INFO] receive update: %+v", update)

			if update.Message == nil {
				log.Printf("[DEBUG] empty message body")
				continue
			}

			msgJSON, _ := json.Marshal(update.Message)
			log.Printf("[DEBUG] %s", string(msgJSON))

			msg := l.transform(update.Message)
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

			if resp, send := l.Bots.OnMessage(*msg); send {
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
	l.Save(l.transform(msg))
}

func (l *TelegramListener) transform(msg *tbapi.Message) *bot.Message {
	message := bot.Message{
		ID:   msg.MessageID,
		Sent: msg.Time(),
		Text: msg.Text,
	}

	if msg.From != nil {
		message.From = bot.User{
			Username:    msg.From.UserName,
			DisplayName: msg.From.FirstName + " " + msg.From.LastName,
		}
	}

	if msg.ReplyToMessage != nil {
		message.ReplyToMessage = l.transform(msg.ReplyToMessage)
	}

	switch {
	case msg.Photo != nil && len(*msg.Photo) > 0:
		sizes := *msg.Photo
		message.Picture = &bot.Picture{
			Class: "photo",
			Image: bot.Image{
				FileID:  sizes[0].FileID,
				Width:   sizes[0].Width,
				Height:  sizes[0].Height,
				Sources: l.convertPhotoSizes(*msg.Photo),
			},
			Caption: msg.Caption,
			Thumbnail: &bot.Source{
				FileID: sizes[0].FileID,
				Width:  sizes[0].Width,
				Height: sizes[0].Height,
				Size:   sizes[0].FileSize,
			},
		}

	case msg.Sticker != nil:
		class, extensionFrom, extensionTo := func(isAnimated bool) (string, string, string) {
			if isAnimated {
				return "animated-sticker", "tgs", "json"
			}
			return "sticker", "webp", "png"
		}(msg.Sticker.IsAnimated)

		message.Picture = &bot.Picture{
			Class: class,
			Image: bot.Image{
				FileID: msg.Sticker.FileID + "." + extensionTo,
				Width:  msg.Sticker.Width,
				Height: msg.Sticker.Height,
				Alt:    msg.Sticker.Emoji,
				Type:   extensionTo,
			},
			Sources: l.convertSticker(*msg.Sticker, extensionFrom, extensionTo),
		}

		if msg.Sticker.Thumbnail != nil {
			message.Picture.Thumbnail = &bot.Source{
				FileID: msg.Sticker.Thumbnail.FileID,
				Width:  msg.Sticker.Thumbnail.Width,
				Height: msg.Sticker.Thumbnail.Height,
			}
		}

	case msg.Animation != nil: // have to be before Document case block, run tests
		message.Animation = &bot.Animation{
			FileID:   msg.Animation.FileID,
			FileName: msg.Animation.FileName,
			Size:     msg.Animation.FileSize,
			MimeType: msg.Animation.MimeType,
			Duration: msg.Animation.Duration,
			Width:    msg.Animation.Width,
			Height:   msg.Animation.Height,
		}

		if msg.Animation.Thumbnail != nil {
			message.Animation.Thumbnail = &bot.Source{
				FileID: msg.Animation.Thumbnail.FileID,
				Width:  msg.Animation.Thumbnail.Width,
				Height: msg.Animation.Thumbnail.Height,
				Size:   msg.Animation.Thumbnail.FileSize,
			}
		}

	case msg.Document != nil:
		message.Document = &bot.Document{
			FileID:   msg.Document.FileID,
			FileName: msg.Document.FileName,
			Size:     msg.Document.FileSize,
			MimeType: msg.Document.MimeType,
			Caption:  msg.Caption,
		}

		if msg.Document.Thumbnail != nil {
			message.Document.Thumbnail = &bot.Source{
				FileID: msg.Document.Thumbnail.FileID,
				Width:  msg.Document.Thumbnail.Width,
				Height: msg.Document.Thumbnail.Height,
				Size:   msg.Document.Thumbnail.FileSize,
			}
		}

	case msg.Voice != nil:
		message.Voice = &bot.Voice{
			Duration: msg.Voice.Duration,
			Sources: []bot.Source{
				{
					FileID: msg.Voice.FileID,
					Type:   msg.Voice.MimeType,
					Size:   msg.Voice.FileSize,
				},
				{
					FileID: msg.Voice.FileID + ".mp3",
					Type:   "audio/mp3",
				},
			},
		}
	}

	return &message
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

func (l *TelegramListener) convertSticker(sticker tbapi.Sticker, extensionFrom string, extensionTo string) []bot.Source {
	var result []bot.Source

	result = append(
		result,
		bot.Source{
			FileID: sticker.FileID,
			Type:   extensionFrom,
			Size:   sticker.FileSize,
		},
	)

	result = append(
		result,
		bot.Source{
			FileID: sticker.FileID + "." + extensionTo,
			Type:   extensionTo,
		},
	)

	return result
}
