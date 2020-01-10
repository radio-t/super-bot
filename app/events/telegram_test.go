package events

import (
	"testing"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/stretchr/testify/assert"

	"github.com/radio-t/gitter-rt-bot/app/bot"
)

func Test_convertTextMessageText(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		bot.Message{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Text: "Message",
		},
		l.convert(
			&tbapi.Message{
				From: &tbapi.User{
					UserName:  "username",
					FirstName: "First",
					LastName:  "Last",
				},
				Date: 1578627415,
				Text: "Message",
			},
		),
	)
}

func Test_convertTextMessagePhoto(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		bot.Message{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Picture: &bot.Picture{
				Caption: "caption",
				Image: bot.Image{
					Source: bot.Source{
						FileID: "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA20AA5C9AgABFgQ",
						Width:  320,
						Height: 149,
					},
					Sources: []bot.Source{
						bot.Source{
							FileID: "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA20AA5C9AgABFgQ",
							Width:  320,
							Height: 149,
							Size:   6262,
						},
						bot.Source{
							FileID: "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA3gAA5G9AgABFgQ",
							Width:  800,
							Height: 373,
							Size:   30240,
						},
						bot.Source{
							FileID: "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA3kAA5K9AgABFgQ",
							Width:  1280,
							Height: 597,
							Size:   55267,
						},
					},
				},
			},
		},
		l.convert(
			&tbapi.Message{
				From: &tbapi.User{
					UserName:  "username",
					FirstName: "First",
					LastName:  "Last",
				},
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
			},
		),
	)
}

func Test_convertTextMessageSticker(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		bot.Message{
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Picture: &bot.Picture{
				Image: bot.Image{
					Source: bot.Source{
						FileID: "AAQCAANYAgACWVMYAAE7DXqIBZjlVYNwmg4ABAEAB20AA9uMAAIWBA.jpg",
						Width:  128,
						Height: 128,
						Alt:    "4️⃣",
					},
				},
				Sources: []bot.Source{
					bot.Source{
						FileID: "AAQCAANYAgACWVMYAAE7DXqIBZjlVYNwmg4ABAEAB20AA9uMAAIWBA",
						Type:   "webp",
						Size:   4766,
					},
					bot.Source{
						FileID: "AAQCAANYAgACWVMYAAE7DXqIBZjlVYNwmg4ABAEAB20AA9uMAAIWBA.jpg",
						Type:   "jpeg",
					},
				},
			},
		},
		l.convert(
			&tbapi.Message{
				From: &tbapi.User{
					UserName:  "username",
					FirstName: "First",
					LastName:  "Last",
				},
				Date: 1578627415,
				Sticker: &tbapi.Sticker{
					FileID:   "CAADAgADWAIAAllTGAABOw16iAWY5VUWBA",
					Width:    512,
					Height:   512,
					FileSize: 23458,
					Thumbnail: &tbapi.PhotoSize{
						FileID:   "AAQCAANYAgACWVMYAAE7DXqIBZjlVYNwmg4ABAEAB20AA9uMAAIWBA",
						Width:    128,
						Height:   128,
						FileSize: 4766,
					},
					Emoji: "4️⃣",
				},
			},
		),
	)
}
