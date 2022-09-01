package events

import (
	"context"
	"testing"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/radio-t/super-bot/app/bot"
)

func TestTelegramListener_DoNoBots(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &mockTbAPI{}
	bots := &bot.InterfaceMock{OnMessageFunc: func(msg bot.Message) bot.Response {
		return bot.Response{Send: false}
	}}

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
	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan))

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")

	assert.Equal(t, 1, len(msgLogger.SaveCalls()))
	assert.Equal(t, "text 123", msgLogger.SaveCalls()[0].Msg.Text)
	assert.Equal(t, "user", msgLogger.SaveCalls()[0].Msg.From.Username)
}

func TestTelegramListener_DoWithBots(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &mockTbAPI{}
	bots := &bot.InterfaceMock{OnMessageFunc: func(msg bot.Message) bot.Response {
		t.Logf("on-message: %+v", msg)
		if msg.Text == "text 123" && msg.From.Username == "user" {
			return bot.Response{Send: true, Text: "bot's answer"}
		}
		return bot.Response{}
	}}

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
	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan))

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "bot's answer"
	})).Return(tbapi.Message{Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	assert.Equal(t, 2, len(msgLogger.SaveCalls()))
	assert.Equal(t, "text 123", msgLogger.SaveCalls()[0].Msg.Text)
	assert.Equal(t, "user", msgLogger.SaveCalls()[0].Msg.From.Username)
	assert.Equal(t, "bot's answer", msgLogger.SaveCalls()[1].Msg.Text)
	tbAPI.AssertNumberOfCalls(t, "Send", 1)
}

func TestTelegramListener_DoWithRtjc(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &mockTbAPI{}
	bots := &bot.InterfaceMock{}

	l := TelegramListener{
		MsgLogger: msgLogger,
		TbAPI:     tbAPI,
		Bots:      bots,
		Group:     "gr",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	tbAPI.On("GetChat", mock.Anything).Return(tbapi.Chat{ID: 123}, nil)

	updChan := make(chan tbapi.Update, 1)

	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan))

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "rtjc message"
	})).Return(tbapi.Message{Text: "rtjc message", From: &tbapi.User{UserName: "user"}}, nil)

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.PinChatMessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.ChatID == 123
	})).Return(tbapi.Message{}, nil)
	time.AfterFunc(time.Millisecond*50, func() {
		assert.NoError(t, l.Submit(ctx, "rtjc message", true))
	})

	err := l.Do(ctx)
	assert.EqualError(t, err, "context deadline exceeded")
	assert.Equal(t, 1, len(msgLogger.SaveCalls()))
	assert.Equal(t, "rtjc message", msgLogger.SaveCalls()[0].Msg.Text)
	tbAPI.AssertNumberOfCalls(t, "Send", 2)
}

func TestTelegramListener_DoWithAutoBan(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &mockTbAPI{}
	bots := &bot.InterfaceMock{OnMessageFunc: func(msg bot.Message) bot.Response {
		return bot.Response{Send: false}
	}}

	l := TelegramListener{
		MsgLogger: msgLogger,
		TbAPI:     tbAPI,
		Bots:      bots,
		Group:     "gr",
		AllActivityTerm: Terminator{
			BanDuration:   100 * time.Millisecond,
			BanPenalty:    3,
			AllowedPeriod: 1 * time.Second,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	now := int(time.Now().Unix())
	updMsg := tbapi.Update{
		Message: &tbapi.Message{
			Chat: &tbapi.Chat{ID: 123},
			Text: "text 123",
			From: &tbapi.User{UserName: "user_name", ID: 1},
			Date: now,
		},
	}

	tbAPI.On("GetChat", mock.Anything).Return(tbapi.Chat{ID: 123}, nil)

	updChan := make(chan tbapi.Update, 5)
	updChan <- updMsg
	updChan <- updMsg
	updChan <- updMsg
	updChan <- updMsg
	updChan <- updMsg
	close(updChan)

	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan))

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "[@user_name](tg://user?id=1) _тебя слишком много, отдохни..._"
	})).Return(tbapi.Message{Text: "[@user_name](tg://user?id=1) _тебя слишком много, отдохни..._", From: &tbapi.User{UserName: "user_name", ID: 1}}, nil).Once()

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.RestrictChatMemberConfig) bool {
		t.Logf("send: %+v", c)
		return c.ChatID == 123
	})).Return(tbapi.Message{}, nil)
	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")

	tbAPI.AssertNumberOfCalls(t, "Send", 2)
	assert.Equal(t, 6, len(msgLogger.SaveCalls()))
	assert.Equal(t, "text 123", msgLogger.SaveCalls()[0].Msg.Text)
	assert.Equal(t, "user_name", msgLogger.SaveCalls()[0].Msg.From.Username)
	assert.Equal(t, "user_name", msgLogger.SaveCalls()[5].Msg.From.Username)
	assert.Equal(t, "[@user_name](tg://user?id=1) _тебя слишком много, отдохни..._", msgLogger.SaveCalls()[4].Msg.Text)
}

func TestTelegramListener_DoWithBotBan(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &mockTbAPI{}
	bots := &bot.InterfaceMock{OnMessageFunc: func(msg bot.Message) bot.Response {
		t.Logf("on-message: %+v", msg)
		if msg.Text == "text 123" && msg.From.Username == "user" {
			return bot.Response{Send: true, Text: "bot's answer", BanInterval: 2 * time.Minute, User: bot.User{Username: "user", ID: 1}}
		}
		return bot.Response{}
	}}

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
			From: &tbapi.User{UserName: "user", ID: 123},
			Date: int(time.Date(2020, 2, 11, 19, 35, 55, 9, time.UTC).Unix()),
		},
	}

	tbAPI.On("GetChat", mock.Anything).Return(tbapi.Chat{ID: 123}, nil)

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan))

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "bot's answer"
	})).Return(tbapi.Message{Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil)

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.RestrictChatMemberConfig) bool {
		t.Logf("send: %+v", c)
		return c.ChatID == 123
	})).Return(tbapi.Message{}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	assert.Equal(t, 2, len(msgLogger.SaveCalls()))
	assert.Equal(t, "text 123", msgLogger.SaveCalls()[0].Msg.Text)
	assert.Equal(t, "user", msgLogger.SaveCalls()[0].Msg.From.Username)
	assert.Equal(t, "bot's answer", msgLogger.SaveCalls()[1].Msg.Text)
	tbAPI.AssertNumberOfCalls(t, "Send", 2)
}

func TestTelegramListener_DoPinMessages(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}

	tbAPI := &mockTbAPI{}
	bots := &bot.InterfaceMock{OnMessageFunc: func(msg bot.Message) bot.Response {
		t.Logf("on-message: %+v", msg)
		if msg.Text == "text 123" && msg.From.Username == "user" {
			return bot.Response{Send: true, Text: "bot's answer", Pin: true}
		}
		return bot.Response{}
	}}

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
	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan))

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "bot's answer"
	})).Return(tbapi.Message{MessageID: 456, Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil)

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.PinChatMessageConfig) bool {
		t.Logf("pin chat message: %+v", c)
		return c.MessageID == 456
	})).Return(tbapi.Message{}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	assert.Equal(t, 1, len(bots.OnMessageCalls()))
	tbAPI.AssertExpectations(t)
}

func TestTelegramListener_DoUnpinMessages(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}

	tbAPI := &mockTbAPI{}
	bots := &bot.InterfaceMock{OnMessageFunc: func(msg bot.Message) bot.Response {
		t.Logf("on-message: %+v", msg)
		if msg.Text == "text 123" && msg.From.Username == "user" {
			return bot.Response{Send: true, Text: "bot's answer", Unpin: true}
		}
		return bot.Response{}
	}}

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
	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan))

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "bot's answer"
	})).Return(tbapi.Message{MessageID: 456, Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil)

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.UnpinChatMessageConfig) bool {
		t.Logf("unpin chat message: %+v", c)
		return c.ChatID == 123
	})).Return(tbapi.Message{}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	assert.Equal(t, 1, len(bots.OnMessageCalls()))
	tbAPI.AssertExpectations(t)
}

func TestTelegramListener_DoNotSaveMessagesFromOtherChats(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &mockTbAPI{}
	bots := &bot.InterfaceMock{OnMessageFunc: func(msg bot.Message) bot.Response {
		return bot.Response{Send: true, Text: "bot's answer"}
	}}

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
			Chat: &tbapi.Chat{ID: 456}, // different group or private message
			Text: "text 123",
			From: &tbapi.User{UserName: "user"},
			Date: int(time.Date(2020, 2, 11, 19, 35, 55, 9, time.UTC).Unix()),
		},
	}

	tbAPI.On("GetChat", mock.Anything).Return(tbapi.Chat{ID: 123}, nil)

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan))

	tbAPI.On("Send", mock.Anything).Return(tbapi.Message{Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")

	assert.Equal(t, 0, len(msgLogger.SaveCalls()))
	assert.Equal(t, 1, len(bots.OnMessageCalls()))
	tbAPI.AssertExpectations(t)
}

func TestTelegram_transformTextMessage(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			ID: 30,
			From: bot.User{
				ID:          100000001,
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent:   time.Unix(1578627415, 0),
			Text:   "Message",
			ChatID: 123456,
		},
		l.transform(
			&tbapi.Message{
				Chat: &tbapi.Chat{
					ID: 123456,
				},
				From: &tbapi.User{
					ID:        100000001,
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
				Photo: []tbapi.PhotoSize{
					{
						FileID:   "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA20AA5C9AgABFgQ",
						Width:    320,
						Height:   149,
						FileSize: 6262,
					},
					{
						FileID:   "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA3gAA5G9AgABFgQ",
						Width:    800,
						Height:   373,
						FileSize: 30240,
					},
					{
						FileID:   "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA3kAA5K9AgABFgQ",
						Width:    1280,
						Height:   597,
						FileSize: 55267,
					},
				},
				Caption: "caption",
				CaptionEntities: []tbapi.MessageEntity{
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
				Entities: []tbapi.MessageEntity{
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
