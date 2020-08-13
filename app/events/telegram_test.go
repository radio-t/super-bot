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

	msgLogger.AssertExpectations(t)
	msgLogger.AssertNumberOfCalls(t, "Save", 1)
}

func TestTelegramListener_DoWithBots(t *testing.T) {
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

func TestTelegramListener_DoWithRtjc(t *testing.T) {
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

	tbAPI.On("GetChat", mock.Anything).Return(tbapi.Chat{ID: 123}, nil)

	updChan := make(chan tbapi.Update, 1)

	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan), nil)

	msgLogger.On("Save", mock.MatchedBy(func(msg *bot.Message) bool {
		t.Logf("save: %+v", msg)
		return msg.Text == "rtjc message"
	}))

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "rtjc message"
	})).Return(tbapi.Message{Text: "rtjc message", From: &tbapi.User{UserName: "user"}}, nil)

	tbAPI.On("PinChatMessage", mock.Anything).Return(tbapi.APIResponse{}, nil)
	time.AfterFunc(time.Millisecond*50, func() {
		l.Submit(ctx, "rtjc message", true)
	})

	err := l.Do(ctx)
	assert.EqualError(t, err, "context deadline exceeded")
	msgLogger.AssertExpectations(t)
	msgLogger.AssertNumberOfCalls(t, "Save", 1)
	tbAPI.AssertNumberOfCalls(t, "Send", 1)
	tbAPI.AssertNumberOfCalls(t, "PinChatMessage", 1)
}

func TestTelegramListener_DoWithAutoBan(t *testing.T) {
	msgLogger := &mockMsgLogger{}
	tbAPI := &mockTbAPI{}
	bots := &bot.MockInterface{}

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

	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan), nil)

	bots.On("OnMessage", mock.Anything).Return(bot.Response{Send: false})
	msgLogger.On("Save", mock.MatchedBy(func(msg *bot.Message) bool {
		t.Logf("%v", msg)
		return msg.Text == "text 123" && msg.From.Username == "user_name"
	}))
	msgLogger.On("Save", mock.MatchedBy(func(msg *bot.Message) bool {
		t.Logf("%v", msg)
		return msg.Text == "[@user_name](tg://user?id=1) _тебя слишком много, отдохни..._" && msg.From.Username == "user_name"
	})).Once()

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "[@user_name](tg://user?id=1) _тебя слишком много, отдохни..._"
	})).Return(tbapi.Message{Text: "[@user_name](tg://user?id=1) _тебя слишком много, отдохни..._", From: &tbapi.User{UserName: "user_name", ID: 1}}, nil).Once()

	tbAPI.On("RestrictChatMember", mock.Anything).Return(tbapi.APIResponse{Ok: true}, nil)
	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")

	tbAPI.AssertNumberOfCalls(t, "Send", 1)
	msgLogger.AssertExpectations(t)
	msgLogger.AssertNumberOfCalls(t, "Save", 6)
}

func TestTelegramListener_DoWithBotBan(t *testing.T) {
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
	})).Return(bot.Response{Send: true, Text: "bot's answer", BanInterval: 2 * time.Minute})

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

	tbAPI.On("RestrictChatMember", mock.Anything).Return(tbapi.APIResponse{Ok: true}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	msgLogger.AssertExpectations(t)
	msgLogger.AssertNumberOfCalls(t, "Save", 2)
	tbAPI.AssertNumberOfCalls(t, "Send", 1)
}

func TestTelegramListener_DoPinMessages(t *testing.T) {
	msgLogger := &mockMsgLogger{}
	msgLogger.On("Save", mock.Anything).Return()

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
	})).Return(bot.Response{Send: true, Text: "bot's answer", Pin: true})

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "bot's answer"
	})).Return(tbapi.Message{MessageID: 456, Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil)

	tbAPI.On("PinChatMessage", mock.MatchedBy(func(c tbapi.PinChatMessageConfig) bool {
		t.Logf("pin chat message: %+v", c)
		return c.MessageID == 456
	})).Return(tbapi.APIResponse{Ok: true}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	bots.AssertExpectations(t)
	tbAPI.AssertExpectations(t)
}

func TestTelegramListener_DoUnpinMessages(t *testing.T) {
	msgLogger := &mockMsgLogger{}
	msgLogger.On("Save", mock.Anything).Return()

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
	})).Return(bot.Response{Send: true, Text: "bot's answer", Unpin: true})

	tbAPI.On("Send", mock.MatchedBy(func(c tbapi.MessageConfig) bool {
		t.Logf("send: %+v", c)
		return c.Text == "bot's answer"
	})).Return(tbapi.Message{MessageID: 456, Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil)

	tbAPI.On("UnpinChatMessage", mock.MatchedBy(func(c tbapi.UnpinChatMessageConfig) bool {
		t.Logf("unpin chat message: %+v", c)
		return c.ChatID == 123
	})).Return(tbapi.APIResponse{Ok: true}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	bots.AssertExpectations(t)
	tbAPI.AssertExpectations(t)
}

func TestTelegramListener_DoNotSaveMessagesFromOtherChats(t *testing.T) {
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
	tbAPI.On("GetUpdatesChan", mock.Anything).Return(tbapi.UpdatesChannel(updChan), nil)

	bots.On("OnMessage", mock.Anything).Return(bot.Response{Send: true, Text: "bot's answer"})
	tbAPI.On("Send", mock.Anything).Return(tbapi.Message{Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil)

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")

	msgLogger.AssertNotCalled(t, "Save", mock.Anything)
	bots.AssertExpectations(t)
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
