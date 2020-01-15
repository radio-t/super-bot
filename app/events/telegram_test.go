package events

import (
	"testing"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/stretchr/testify/assert"

	"github.com/radio-t/gitter-rt-bot/app/bot"
)

func Test_convertTextMessage(t *testing.T) {
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

func Test_convertPhoto(t *testing.T) {
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
				Class:   "photo",
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

func Test_convertSticker(t *testing.T) {
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
				Class: "sticker",
				Image: bot.Image{
					Source: bot.Source{
						FileID: "CAADAgADWAIAAllTGAABOw16iAWY5VUWBA.png",
						Width:  512,
						Height: 512,
						Alt:    "4️⃣",
					},
				},
				Sources: []bot.Source{
					bot.Source{
						FileID: "CAADAgADWAIAAllTGAABOw16iAWY5VUWBA",
						Type:   "webp",
						Size:   23458,
					},
					bot.Source{
						FileID: "CAADAgADWAIAAllTGAABOw16iAWY5VUWBA.png",
						Type:   "png",
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
					FileID:     "CAADAgADWAIAAllTGAABOw16iAWY5VUWBA",
					Width:      512,
					Height:     512,
					FileSize:   23458,
					IsAnimated: false,
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

func Test_convertAnimatedSticker(t *testing.T) {
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
				Class: "animated-sticker",
				Image: bot.Image{
					Source: bot.Source{
						FileID: "CAADAgAD8gEAArD72weo9_9Bp6KNxxYE.json",
						Width:  512,
						Height: 512,
						Alt:    "👻",
					},
				},
				Sources: []bot.Source{
					bot.Source{
						FileID: "CAADAgAD8gEAArD72weo9_9Bp6KNxxYE",
						Type:   "tgs",
						Size:   2278,
					},
					bot.Source{
						FileID: "CAADAgAD8gEAArD72weo9_9Bp6KNxxYE.json",
						Type:   "json",
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
					FileID:     "CAADAgAD8gEAArD72weo9_9Bp6KNxxYE",
					Width:      512,
					Height:     512,
					FileSize:   2278,
					IsAnimated: true,
					Thumbnail: &tbapi.PhotoSize{
						FileID:   "AAQCAAPyAQACsPvbB6j3_0Gnoo3HRAq4DwAEAQAHbQADGWMAAhYE",
						Width:    128,
						Height:   128,
						FileSize: 2604,
					},
					Emoji: "👻",
				},
			},
		),
	)
}
