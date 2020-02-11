package events

import (
	"context"
	"testing"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/radio-t/super-bot/app/bot"
)

func TestTelegramListener_DoNoBots(t *testing.T) {
	msgLogger := &mockMsgLogger{}
	tbAPI := &mockTbAPI{}
	bots := &bot.MockInterface{}

	l := TelegramListener{
		MsgLogger: msgLogger,
		TbAPI:     tbAPI,
		Bots:      bots,
		Group:     "gr",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	updMsg := tbapi.Update{
		Message: &tbapi.Message{
			Chat: &tbapi.Chat{ID: 123},
			Text: "text 123",
			From: &tbapi.User{UserName: "user"},
		},
	}

	tbAPI.On("GetChat", mock.Anything).Return(tbapi.Chat{ID: 123}, nil)

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan), nil)

	bots.On("OnMessage", mock.Anything).Return(bot.Response{Send: false})
	msgLogger.On("Save", mock.MatchedBy(func(msg *bot.Message) bool {
		t.Logf("%v", msg)
		return msg.Text == "text 123" && msg.From.Username == "user"
	}))
	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
}

func TestTelegramListener_WithBots(t *testing.T) {
	msgLogger := &mockMsgLogger{}
	tbAPI := &mockTbAPI{}
	bots := &bot.MockInterface{}

	l := TelegramListener{
		MsgLogger: msgLogger,
		TbAPI:     tbAPI,
		Bots:      bots,
		Group:     "gr",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Minute)
	defer cancel()

	updMsg := tbapi.Update{
		Message: &tbapi.Message{
			Chat: &tbapi.Chat{ID: 123},
			Text: "text 123",
			From: &tbapi.User{UserName: "user"},
			Date: int(time.Date(2020, 2, 11, 19, 35, 55, 9, time.UTC).Unix()),
		},
	}

	tbAPI.On("GetChat", mock.Anything).Return(tbapi.Chat{ID: 123}, nil)

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan), nil)

	bots.On("OnMessage", mock.MatchedBy(func(msg bot.Message) bool {
		t.Logf("on-message: %+v", msg)
		return msg.Text == "text 123" && msg.From.Username == "user"
	})).Return(bot.Response{Send: true, Text: "bot's answer"})

	msgLogger.On("Save", mock.MatchedBy(func(msg *bot.Message) bool {
		t.Logf("save: %+v", msg)
		return msg.Text == "text 123" && msg.From.Username == "user"
	}))
	msgLogger.On("Save", mock.MatchedBy(func(msg *bot.Message) bool {
		t.Logf("save: %+v", msg)
		return msg.Text == "bot's answer"
	}))

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "bot's answer"
	})).Return(tbapi.Message{Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	msgLogger.AssertExpectations(t)
	msgLogger.AssertNumberOfCalls(t, "Save", 2)
	tbAPI.AssertNumberOfCalls(t, "Send", 1)

}

func TestTelegram_transformTextMessage(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			ID: 30,
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Text: "Message",
		},
		l.transform(
			&tbapi.Message{
				From: &tbapi.User{
					UserName:  "username",
					FirstName: "First",
					LastName:  "Last",
				},
				MessageID: 30,
				Date:      1578627415,
				Text:      "Message",
			},
		),
	)
}

func TestTelegram_transformPhoto(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Image: &bot.Image{
				FileID:  "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA3kAA5K9AgABFgQ",
				Width:   1280,
				Height:  597,
				Caption: "caption",
				Entities: &[]bot.Entity{
					{
						Type:   "bold",
						Offset: 0,
						Length: 7,
					},
				},
			},
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				Photo: &[]tbapi.PhotoSize{
					tbapi.PhotoSize{
						FileID:   "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA20AA5C9AgABFgQ",
						Width:    320,
						Height:   149,
						FileSize: 6262,
					},
					tbapi.PhotoSize{
						FileID:   "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA3gAA5G9AgABFgQ",
						Width:    800,
						Height:   373,
						FileSize: 30240,
					},
					tbapi.PhotoSize{
						FileID:   "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA3kAA5K9AgABFgQ",
						Width:    1280,
						Height:   597,
						FileSize: 55267,
					},
				},
				Caption: "caption",
				CaptionEntities: &[]tbapi.MessageEntity{
					{
						Type:   "bold",
						Offset: 0,
						Length: 7,
					},
				},
			},
		),
	)
}

func TestTelegram_transformEntities(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Text: "@username тебя слишком много, отдохни...",
			Entities: &[]bot.Entity{
				{
					Type:   "mention",
					Offset: 0,
					Length: 9,
				},
				{
					Type:   "italic",
					Offset: 10,
					Length: 30,
				},
			},
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				Text: "@username тебя слишком много, отдохни...",
				Entities: &[]tbapi.MessageEntity{
					{
						Type:   "mention",
						Offset: 0,
						Length: 9,
					},
					{
						Type:   "italic",
						Offset: 10,
						Length: 30,
					},
				},
			},
		),
	)
}
