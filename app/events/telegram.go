package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

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

			msgJSON, err := json.Marshal(update.Message)
			if err != nil {
				log.Printf("[ERROR] failed to marshal update.Message to JSON: %v", err)
			} else {
				log.Printf("[DEBUG] %s", string(msgJSON))
			}

			msg := l.transform(update.Message)
			l.Save(msg) // save to report

			log.Printf("[DEBUG] incoming msg: %+v", msg)

			// check for ban
			if b := l.check(msg.From); b.active {
				if b.new {
					m := fmt.Sprintf("@%s _тебя слишком много, отдохни..._", msg.From.Username)
					tbMsg := tbapi.NewMessage(update.Message.Chat.ID, m)
					tbMsg.ParseMode = tbapi.ModeMarkdown
					if res, err := l.botAPI.Send(tbMsg); err != nil {
						log.Printf("[WARN] failed to send, %v", err)
					} else {
						l.saveBotMessage(&res)
					}

					if err := l.banUser(update.Message.Chat.ID, update.Message.From.ID); err != nil {
						log.Printf("[ERROR] failed to ban user %v: %v", msg.From, err)
					}
				}
				continue
			}

			if resp, send := l.Bots.OnMessage(*msg); send {
				log.Printf("[DEBUG] bot response - %+v", resp)
				tbMsg := tbapi.NewMessage(update.Message.Chat.ID, resp)
				tbMsg.ParseMode = tbapi.ModeMarkdown
				if res, err := l.botAPI.Send(tbMsg); err != nil {
					log.Printf("[WARN] can't send tbMsg to telegram, %v", err)
				} else {
					l.saveBotMessage(&res)
				}
			}

		case msg := <-l.msgs.ch: // publish messages from outside clients
			chat, err := l.botAPI.GetChat(tbapi.ChatConfig{SuperGroupUsername: l.GroupID})
			if err != nil {
				return errors.Wrapf(err, "can't get chat for %s", l.GroupID)
			}
			tbMsg := tbapi.NewMessage(chat.ID, msg)
			tbMsg.ParseMode = tbapi.ModeMarkdown
			if res, err := l.botAPI.Send(tbMsg); err != nil {
				log.Printf("[WARN] can't send msg to telegram, %v", err)
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

// The bot must be an administrator in the supergroup for this to work
// and must have the appropriate admin rights.
func (l *TelegramListener) banUser(chatID int64, userID int) error {
	banDuration := l.AllowedPeriod

	// From Telegram Bot API documentation:
	// If user is restricted for more than 366 days or less than 30 seconds from the current time,
	// they are considered to be restricted forever
	if banDuration < 30*time.Second {
		banDuration = 1 * time.Minute
	}

	resp, err := l.botAPI.RestrictChatMember(tbapi.RestrictChatMemberConfig{
		ChatMemberConfig: tbapi.ChatMemberConfig{
			ChatID: chatID,
			UserID: userID,
		},
		UntilDate:             time.Now().Add(banDuration).Unix(),
		CanSendMessages:       new(bool),
		CanSendMediaMessages:  new(bool),
		CanSendOtherMessages:  new(bool),
		CanAddWebPagePreviews: new(bool),
	})
	if err != nil {
		return err
	}

	if !resp.Ok {
		return fmt.Errorf("response is not Ok: %v", string(resp.Result))
	}

	return nil
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

	if msg.ForwardFrom != nil {
		message.ForwardFrom = &bot.User{
			Username:    msg.ForwardFrom.UserName,
			DisplayName: msg.ForwardFrom.FirstName + " " + msg.ForwardFrom.LastName,
		}
		message.ForwardFromMessageID = msg.ForwardFromMessageID
	}

	if msg.ForwardFromChat != nil {
		message.ForwardFromChat = &bot.Chat{
			ID:        msg.ForwardFromChat.ID,
			Type:      msg.ForwardFromChat.Type,
			Title:     msg.ForwardFromChat.Title,
			UserName:  msg.ForwardFromChat.UserName,
			FirstName: msg.ForwardFromChat.FirstName,
			LastName:  msg.ForwardFromChat.LastName,
		}
		message.ForwardFromMessageID = msg.ForwardFromMessageID
	}

	if msg.ReplyToMessage != nil {
		message.ReplyToMessage = l.transform(msg.ReplyToMessage)
	}

	switch {
	case msg.Entities != nil && len(*msg.Entities) > 0:
		var entities []bot.Entity
		for _, entity := range *msg.Entities {
			e := bot.Entity{
				Type:   entity.Type,
				Offset: entity.Offset,
				Length: entity.Length,
				URL:    entity.URL,
			}
			if entity.User != nil {
				e.User = &bot.User{
					Username:    entity.User.UserName,
					DisplayName: entity.User.FirstName + " " + entity.User.LastName,
				}
			}
			entities = append(entities, e)
		}
		message.Entities = &entities

	case msg.Photo != nil && len(*msg.Photo) > 0:
		sizes := *msg.Photo
		message.Picture = &bot.Picture{
			Class: "photo",
			Image: bot.Image{
				FileID:  sizes[0].FileID,
				Width:   sizes[0].Width,
				Height:  sizes[0].Height,
				Sources: l.transformPhotoSizes(*msg.Photo),
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
			Sources: l.transformSticker(*msg.Sticker, extensionFrom, extensionTo),
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

	case msg.Video != nil:
		message.Video = &bot.Video{
			FileID:   msg.Video.FileID,
			Size:     msg.Video.FileSize,
			MimeType: msg.Video.MimeType,
			Duration: msg.Video.Duration,
			Width:    msg.Video.Width,
			Height:   msg.Video.Height,
			Caption:  msg.Caption,
		}

		if msg.Video.Thumbnail != nil {
			message.Video.Thumbnail = &bot.Source{
				FileID: msg.Video.Thumbnail.FileID,
				Width:  msg.Video.Thumbnail.Width,
				Height: msg.Video.Thumbnail.Height,
				Size:   msg.Video.Thumbnail.FileSize,
			}
		}

	case msg.NewChatMembers != nil:
		var users []bot.User
		for _, user := range *msg.NewChatMembers {
			users = append(users, bot.User{
				Username:    user.UserName,
				DisplayName: user.FirstName + " " + user.LastName,
			})
		}

		message.From = bot.User{}
		message.NewChatMembers = &users

	case msg.LeftChatMember != nil:
		message.From = bot.User{}
		message.LeftChatMember = &bot.User{
			Username:    msg.LeftChatMember.UserName,
			DisplayName: msg.LeftChatMember.FirstName + " " + msg.LeftChatMember.LastName,
		}
	}

	return &message
}

func (l *TelegramListener) transformPhotoSizes(sizes []tbapi.PhotoSize) []bot.Source {
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

func (l *TelegramListener) transformSticker(sticker tbapi.Sticker, extensionFrom string, extensionTo string) []bot.Source {
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
