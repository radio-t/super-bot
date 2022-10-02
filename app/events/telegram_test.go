package events

import (
	"context"
	"testing"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/radio-t/super-bot/app/bot"
	"github.com/stretchr/testify/assert"
)

func TestTelegramListener_DoNoBots(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &tbAPIMock{GetChatFunc: func(config tbapi.ChatInfoConfig) (tbapi.Chat, error) {
		return tbapi.Chat{ID: 123}, nil
	}}
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

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.GetUpdatesChanFunc = func(config tbapi.UpdateConfig) tbapi.UpdatesChannel { return updChan }

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")

	assert.Equal(t, 1, len(msgLogger.SaveCalls()))
	assert.Equal(t, "text 123", msgLogger.SaveCalls()[0].Msg.Text)
	assert.Equal(t, "user", msgLogger.SaveCalls()[0].Msg.From.Username)
}

func TestTelegramListener_DoWithBots(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &tbAPIMock{
		GetChatFunc: func(config tbapi.ChatInfoConfig) (tbapi.Chat, error) {
			return tbapi.Chat{ID: 123}, nil
		},
		SendFunc: func(c tbapi.Chattable) (tbapi.Message, error) {
			return tbapi.Message{Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil
		},
	}
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

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.GetUpdatesChanFunc = func(config tbapi.UpdateConfig) tbapi.UpdatesChannel { return updChan }

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	assert.Equal(t, 2, len(msgLogger.SaveCalls()))
	assert.Equal(t, "text 123", msgLogger.SaveCalls()[0].Msg.Text)
	assert.Equal(t, "user", msgLogger.SaveCalls()[0].Msg.From.Username)
	assert.Equal(t, "bot's answer", msgLogger.SaveCalls()[1].Msg.Text)
	assert.Equal(t, 1, len(tbAPI.SendCalls()))
	assert.Equal(t, "bot's answer", tbAPI.SendCalls()[0].C.(tbapi.MessageConfig).Text)
}

func TestTelegramListener_DoWithRtjc(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &tbAPIMock{
		GetChatFunc: func(config tbapi.ChatInfoConfig) (tbapi.Chat, error) {
			return tbapi.Chat{ID: 123}, nil
		},
		SendFunc: func(c tbapi.Chattable) (tbapi.Message, error) {
			switch c.(type) {
			case tbapi.MessageConfig:
				if c.(tbapi.MessageConfig).Text == "rtjc message" {
					return tbapi.Message{Text: "rtjc message", From: &tbapi.User{UserName: "user"}}, nil
				}
			}
			return tbapi.Message{}, nil
		},
	}
	bots := &bot.InterfaceMock{}

	l := TelegramListener{
		MsgLogger: msgLogger,
		TbAPI:     tbAPI,
		Bots:      bots,
		Group:     "gr",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	updChan := make(chan tbapi.Update, 1)

	tbAPI.GetUpdatesChanFunc = func(config tbapi.UpdateConfig) tbapi.UpdatesChannel { return updChan }

	time.AfterFunc(time.Millisecond*50, func() {
		assert.NoError(t, l.Submit(ctx, "rtjc message", true))
	})

	err := l.Do(ctx)
	assert.EqualError(t, err, "context deadline exceeded")
	assert.Equal(t, 1, len(msgLogger.SaveCalls()))
	assert.Equal(t, "rtjc message", msgLogger.SaveCalls()[0].Msg.Text)
	assert.Equal(t, 2, len(tbAPI.SendCalls()))
	assert.Equal(t, int64(123), tbAPI.SendCalls()[1].C.(tbapi.PinChatMessageConfig).ChatID)
}

func TestTelegramListener_DoWithAutoBan(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	first := true
	tbAPI := &tbAPIMock{
		GetChatFunc: func(config tbapi.ChatInfoConfig) (tbapi.Chat, error) {
			return tbapi.Chat{ID: 123}, nil
		},
		SendFunc: func(c tbapi.Chattable) (tbapi.Message, error) {
			return tbapi.Message{Text: "[@user_name](tg://user?id=1) _тебя слишком много, отдохни..._", From: &tbapi.User{UserName: "user_name", ID: 1}}, nil
		},
		RequestFunc: func(c tbapi.Chattable) (*tbapi.APIResponse, error) {
			if first {
				first = false
				return &tbapi.APIResponse{Ok: false}, nil
			}
			return &tbapi.APIResponse{Ok: true}, nil
		},
	}
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

	updChan := make(chan tbapi.Update, 5)
	updChan <- updMsg
	updChan <- updMsg
	updChan <- updMsg
	updChan <- updMsg
	updChan <- updMsg
	close(updChan)

	tbAPI.GetUpdatesChanFunc = func(config tbapi.UpdateConfig) tbapi.UpdatesChannel { return updChan }

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")

	assert.Equal(t, 1, len(tbAPI.SendCalls()))
	assert.Equal(t, "[@user_name](tg://user?id=1) _тебя слишком много, отдохни..._", tbAPI.SendCalls()[0].C.(tbapi.MessageConfig).Text)
	assert.Equal(t, 1, len(tbAPI.RequestCalls()))
	assert.Equal(t, int64(123), tbAPI.RequestCalls()[0].C.(tbapi.RestrictChatMemberConfig).ChatID)
	assert.Equal(t, 6, len(msgLogger.SaveCalls()))
	assert.Equal(t, "text 123", msgLogger.SaveCalls()[0].Msg.Text)
	assert.Equal(t, "user_name", msgLogger.SaveCalls()[0].Msg.From.Username)
	assert.Equal(t, "user_name", msgLogger.SaveCalls()[5].Msg.From.Username)
	assert.Equal(t, "[@user_name](tg://user?id=1) _тебя слишком много, отдохни..._", msgLogger.SaveCalls()[4].Msg.Text)
}

func TestTelegramListener_DoWithBotBan(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &tbAPIMock{
		GetChatFunc: func(config tbapi.ChatInfoConfig) (tbapi.Chat, error) {
			return tbapi.Chat{ID: 123}, nil
		},
		SendFunc: func(c tbapi.Chattable) (tbapi.Message, error) {
			return tbapi.Message{Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil
		},
		RequestFunc: func(c tbapi.Chattable) (*tbapi.APIResponse, error) {
			return &tbapi.APIResponse{}, nil
		},
	}
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

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.GetUpdatesChanFunc = func(config tbapi.UpdateConfig) tbapi.UpdatesChannel { return updChan }

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	assert.Equal(t, 2, len(msgLogger.SaveCalls()))
	assert.Equal(t, "text 123", msgLogger.SaveCalls()[0].Msg.Text)
	assert.Equal(t, "user", msgLogger.SaveCalls()[0].Msg.From.Username)
	assert.Equal(t, "bot's answer", msgLogger.SaveCalls()[1].Msg.Text)
	assert.Equal(t, 1, len(tbAPI.SendCalls()))
	assert.Equal(t, "bot's answer", tbAPI.SendCalls()[0].C.(tbapi.MessageConfig).Text)
	assert.Equal(t, 1, len(tbAPI.RequestCalls()))
	assert.Equal(t, int64(123), tbAPI.RequestCalls()[0].C.(tbapi.RestrictChatMemberConfig).ChatID)
}

func TestTelegramListener_DoPinMessages(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}

	tbAPI := &tbAPIMock{
		GetChatFunc: func(config tbapi.ChatInfoConfig) (tbapi.Chat, error) {
			return tbapi.Chat{ID: 123}, nil
		},
		SendFunc: func(c tbapi.Chattable) (tbapi.Message, error) {
			switch c.(type) {
			case tbapi.MessageConfig:
				if c.(tbapi.MessageConfig).Text == "bot's answer" {
					return tbapi.Message{MessageID: 456, Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil
				}
			}
			return tbapi.Message{}, nil
		},
	}
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

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.GetUpdatesChanFunc = func(config tbapi.UpdateConfig) tbapi.UpdatesChannel { return updChan }

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	assert.Equal(t, 1, len(bots.OnMessageCalls()))
	assert.Equal(t, 2, len(tbAPI.SendCalls()))
	assert.Equal(t, 456, tbAPI.SendCalls()[1].C.(tbapi.PinChatMessageConfig).MessageID)
}

func TestTelegramListener_DoUnpinMessages(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}

	tbAPI := &tbAPIMock{
		GetChatFunc: func(config tbapi.ChatInfoConfig) (tbapi.Chat, error) {
			return tbapi.Chat{ID: 123}, nil
		},
		SendFunc: func(c tbapi.Chattable) (tbapi.Message, error) {
			switch c.(type) {
			case tbapi.MessageConfig:
				if c.(tbapi.MessageConfig).Text == "bot's answer" {
					return tbapi.Message{Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil
				}
			}
			return tbapi.Message{}, nil
		},
	}
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

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.GetUpdatesChanFunc = func(config tbapi.UpdateConfig) tbapi.UpdatesChannel { return updChan }

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")
	assert.Equal(t, 1, len(bots.OnMessageCalls()))
	assert.Equal(t, 2, len(tbAPI.SendCalls()))
	assert.Equal(t, int64(123), tbAPI.SendCalls()[1].C.(tbapi.UnpinChatMessageConfig).ChatID)
}

func TestTelegramListener_DoNotSaveMessagesFromOtherChats(t *testing.T) {
	msgLogger := &msgLoggerMock{SaveFunc: func(msg *bot.Message) { return }}
	tbAPI := &tbAPIMock{
		GetChatFunc: func(config tbapi.ChatInfoConfig) (tbapi.Chat, error) {
			return tbapi.Chat{ID: 123}, nil
		},
		SendFunc: func(c tbapi.Chattable) (tbapi.Message, error) {
			return tbapi.Message{Text: "bot's answer", From: &tbapi.User{UserName: "user"}}, nil
		},
	}

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

	updChan := make(chan tbapi.Update, 1)
	updChan <- updMsg
	close(updChan)
	tbAPI.GetUpdatesChanFunc = func(config tbapi.UpdateConfig) tbapi.UpdatesChannel { return updChan }

	err := l.Do(ctx)
	assert.EqualError(t, err, "telegram update chan closed")

	assert.Equal(t, 0, len(msgLogger.SaveCalls()))
	assert.Equal(t, 1, len(bots.OnMessageCalls()))
	assert.Equal(t, 1, len(tbAPI.SendCalls()))
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
