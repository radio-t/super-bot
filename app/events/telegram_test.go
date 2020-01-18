package events

import (
	"testing"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/stretchr/testify/assert"

	"github.com/radio-t/gitter-rt-bot/app/bot"
)

func Test_transformTextMessage(t *testing.T) {
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

func Test_transformTextMessageWithReply(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			ID: 31,
			From: bot.User{
				Username:    "username",
				DisplayName: "First Last",
			},
			Sent: time.Unix(1578627415, 0),
			Text: "Reply",
			ReplyToMessage: &bot.Message{
				ID: 30,
				From: bot.User{
					Username:    "username2",
					DisplayName: "First2 Last2",
				},
				Sent: time.Unix(1578627415, 0),
				Text: "Message",
			},
		},
		l.transform(
			&tbapi.Message{
				MessageID: 31,
				From: &tbapi.User{
					UserName:  "username",
					FirstName: "First",
					LastName:  "Last",
				},
				Date: 1578627415,
				Text: "Reply",
				ReplyToMessage: &tbapi.Message{
					MessageID: 30,
					From: &tbapi.User{
						UserName:  "username2",
						FirstName: "First2",
						LastName:  "Last2",
					},
					Date: 1578627415,
					Text: "Message",
				},
			},
		),
	)
}

func Test_transformPhoto(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Picture: &bot.Picture{
				Class:   "photo",
				Caption: "caption",
				Image: bot.Image{
					FileID: "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA20AA5C9AgABFgQ",
					Width:  320,
					Height: 149,
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
				Thumbnail: &bot.Source{
					FileID: "AgADAgADFKwxG8r0qUiQByxwp9Gi4s1qwQ8ABAEAAwIAA20AA5C9AgABFgQ",
					Width:  320,
					Height: 149,
					Size:   6262,
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
			},
		),
	)
}

func Test_transformSticker(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Picture: &bot.Picture{
				Class: "sticker",
				Image: bot.Image{
					FileID: "CAADAgADWAIAAllTGAABOw16iAWY5VUWBA.png",
					Width:  512,
					Height: 512,
					Alt:    "4️⃣",
					Type:   "png",
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
				Thumbnail: &bot.Source{
					FileID: "AAQCAANYAgACWVMYAAE7DXqIBZjlVYNwmg4ABAEAB20AA9uMAAIWBA",
					Width:  128,
					Height: 128,
				},
			},
		},
		l.transform(
			&tbapi.Message{
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

func Test_transformAnimatedSticker(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Picture: &bot.Picture{
				Class: "animated-sticker",
				Image: bot.Image{
					FileID: "CAADAgAD8gEAArD72weo9_9Bp6KNxxYE.json",
					Width:  512,
					Height: 512,
					Type:   "json",
					Alt:    "👻",
				},
				Thumbnail: &bot.Source{
					FileID: "AAQCAAPyAQACsPvbB6j3_0Gnoo3HRAq4DwAEAQAHbQADGWMAAhYE",
					Width:  128,
					Height: 128,
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
		l.transform(
			&tbapi.Message{
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

func Test_transformDocument(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Document: &bot.Document{
				FileID:   "BQADAgADlgQAAsyxCUlsinA_gGRZlhYE",
				FileName: "junk-mail.jpg",
				MimeType: "image/jpeg",
				Size:     101780,
				Caption:  "document caption",
				Thumbnail: &bot.Source{
					FileID: "AAQCAAOWBAACzLEJSWyKcD-AZFmWgsfLDgAEAQAHbQADsA4AAhYE",
					Width:  300,
					Height: 300,
					Size:   49827,
				},
			},
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				Document: &tbapi.Document{
					FileID:   "BQADAgADlgQAAsyxCUlsinA_gGRZlhYE",
					FileName: "junk-mail.jpg",
					MimeType: "image/jpeg",
					FileSize: 101780,
					Thumbnail: &tbapi.PhotoSize{
						FileID:   "AAQCAAOWBAACzLEJSWyKcD-AZFmWgsfLDgAEAQAHbQADsA4AAhYE",
						Width:    300,
						Height:   300,
						FileSize: 49827,
					},
				},
				Caption: "document caption",
			},
		),
	)
}

func Test_transformAnimation(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Animation: &bot.Animation{
				FileID:   "CgADBAADBQADZHZtUX7GwEE7RarSFgQ",
				FileName: "giphy.mp4",
				MimeType: "video/mp4",
				Size:     199710,
				Duration: 2,
				Width:    480,
				Height:   266,
				Thumbnail: &bot.Source{
					FileID: "AAQEAAMFAANkdm1RfsbAQTtFqtKNzqcaAAQBAAdzAANzFgACFgQ",
					Width:  90,
					Height: 50,
					Size:   2158,
				},
			},
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				Animation: &tbapi.ChatAnimation{
					FileID:   "CgADBAADBQADZHZtUX7GwEE7RarSFgQ",
					FileName: "giphy.mp4",
					MimeType: "video/mp4",
					FileSize: 199710,
					Duration: 2,
					Width:    480,
					Height:   266,
					Thumbnail: &tbapi.PhotoSize{
						FileID:   "AAQEAAMFAANkdm1RfsbAQTtFqtKNzqcaAAQBAAdzAANzFgACFgQ",
						Width:    90,
						Height:   50,
						FileSize: 2158,
					},
				},
				// no idea why Document is almost copy of Animation
				// unless it's to support some old clients
				Document: &tbapi.Document{
					FileID:   "CgADBAADBQADZHZtUX7GwEE7RarSFgQ",
					FileName: "giphy.mp4",
					MimeType: "video/mp4",
					FileSize: 199710,
					Thumbnail: &tbapi.PhotoSize{
						FileID:   "AAQEAAMFAANkdm1RfsbAQTtFqtKNzqcaAAQBAAdzAANzFgACFgQ",
						Width:    90,
						Height:   50,
						FileSize: 2158,
					},
				},
			},
		),
	)
}

func Test_transformVoice(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Voice: &bot.Voice{
				Duration: 1,
				Sources: []bot.Source{
					{
						FileID: "AwADAgADSAUAAkmeEUnZa0L9pIG5ZBYE",
						Type:   "audio/ogg",
						Size:   6394,
					},
					{
						FileID: "AwADAgADSAUAAkmeEUnZa0L9pIG5ZBYE.mp3",
						Type:   "audio/mp3",
					},
				},
			},
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				Voice: &tbapi.Voice{
					FileID:   "AwADAgADSAUAAkmeEUnZa0L9pIG5ZBYE",
					MimeType: "audio/ogg",
					FileSize: 6394,
					Duration: 1,
				},
			},
		),
	)
}

func Test_transformVideo(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Video: &bot.Video{
				FileID:   "BAADAgADTAUAAkmeEUlD7g0Wpe4cxRYE",
				Width:    240,
				Height:   240,
				Duration: 1,
				MimeType: "video/mp4",
				Size:     71665,
				Caption:  "Caption",
				Thumbnail: &bot.Source{
					FileID: "AAQCAANMBQACSZ4RSUPuDRal7hzFz-rdDwAEAQAHbQADxAgAAhYE",
					Width:  240,
					Height: 240,
					Size:   11733,
				},
			},
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				Video: &tbapi.Video{
					FileID:   "BAADAgADTAUAAkmeEUlD7g0Wpe4cxRYE",
					Width:    240,
					Height:   240,
					Duration: 1,
					Thumbnail: &tbapi.PhotoSize{
						FileID:   "AAQCAANMBQACSZ4RSUPuDRal7hzFz-rdDwAEAQAHbQADxAgAAhYE",
						Width:    240,
						Height:   240,
						FileSize: 11733,
					},
					MimeType: "video/mp4",
					FileSize: 71665,
				},
				Caption: "Caption",
			},
		),
	)
}

func Test_transformForwardFromChat(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Text: "И под конец рабочей недели <...> https://techcrunch.com/2020/01/17/digitalocean-layoffs/",
			ForwardFromChat: &bot.Chat{
				ID:        -1001005993407,
				Type:      "channel",
				Title:     "addmeto",
				UserName:  "addmeto",
				FirstName: "",
				LastName:  "",
			},
			ForwardFromMessageID: 2956,
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				Text: "И под конец рабочей недели <...> https://techcrunch.com/2020/01/17/digitalocean-layoffs/",
				ForwardFromChat: &tbapi.Chat{
					ID:                  -1001005993407,
					Type:                "channel",
					Title:               "addmeto",
					UserName:            "addmeto",
					FirstName:           "",
					LastName:            "",
					AllMembersAreAdmins: false,
					Photo:               nil,
					PinnedMessage:       nil,
				},
				ForwardFromMessageID: 2956,
				ForwardDate:          1579293863,
				Entities: &[]tbapi.MessageEntity{
					{
						Type:   "url",
						Offset: 33,
						Length: 55,
						URL:    "",
						User:   nil,
					},
				},
			},
		),
	)
}

func Test_transformForwardFrom(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			Text: "Я имел ввиду GKE и EKS, да",
			ForwardFrom: &bot.User{
				Username:    "chuhlomin",
				DisplayName: "Konstantin Chukhlomin",
			},
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				Text: "Я имел ввиду GKE и EKS, да",
				ForwardFrom: &tbapi.User{
					ID:           4885399,
					UserName:     "chuhlomin",
					FirstName:    "Konstantin",
					LastName:     "Chukhlomin",
					LanguageCode: "",
					IsBot:        false,
				},
				ForwardDate: 1579362361,
			},
		),
	)
}

func Test_transformUserJoined(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			NewChatMembers: &[]bot.User{
				{
					Username:    "username1",
					DisplayName: "First1 Last1",
				},
				{
					Username:    "username2",
					DisplayName: "First2 Last2",
				},
			},
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				From: &tbapi.User{
					FirstName: "First1",
					LastName:  "Last1",
					UserName:  "username1",
				},
				NewChatMembers: &[]tbapi.User{
					{
						FirstName: "First1",
						LastName:  "Last1",
						UserName:  "username1",
					},
					{
						FirstName: "First2",
						LastName:  "Last2",
						UserName:  "username2",
					},
				},
			},
		),
	)
}

func Test_transformUserLeft(t *testing.T) {
	l := TelegramListener{}
	assert.Equal(
		t,
		&bot.Message{
			Sent: time.Unix(1578627415, 0),
			LeftChatMember: &bot.User{
				Username:    "username1",
				DisplayName: "First1 Last1",
			},
		},
		l.transform(
			&tbapi.Message{
				Date: 1578627415,
				From: &tbapi.User{
					FirstName: "First1",
					LastName:  "Last1",
					UserName:  "username1",
				},
				LeftChatMember: &tbapi.User{
					FirstName: "First1",
					LastName:  "Last1",
					UserName:  "username1",
				},
			},
		),
	)
}
