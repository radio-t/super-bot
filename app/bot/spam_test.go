package bot

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestNewSpamFilter(t *testing.T) {
	client := &mocks.HTTPClient{}
	sf := NewSpamFilter("http://localhost", client, false)
	assert.NotNil(t, sf)
	assert.Equal(t, "http://localhost", sf.casAPI)
	assert.Equal(t, client, sf.client)
	assert.False(t, sf.dry)
}

func TestSpamFilter_OnMessage(t *testing.T) {
	tests := []struct {
		name           string
		mockResp       string
		mockStatusCode int
		dryMode        bool
		expectedResp   Response
	}{
		{
			name:           "User is not a spammer",
			mockResp:       `{"ok": false, "description": "Not a spammer"}`,
			mockStatusCode: 200,
			dryMode:        false,
			expectedResp:   Response{},
		},
		{
			name:           "User is a spammer, not in dry mode",
			mockResp:       `{"ok": true, "description": "Is a spammer"}`,
			mockStatusCode: 200,
			dryMode:        false,
			expectedResp: Response{
				Text:          "this is spam! go to ban, Test User",
				Send:          true,
				ReplyTo:       1,
				BanInterval:   permanentBanDuration,
				DeleteReplyTo: true,
			},
		},
		{
			name:           "User is a spammer, in dry mode",
			mockResp:       `{"ok": true, "description": "Is a spammer"}`,
			mockStatusCode: 200,
			dryMode:        true,
			expectedResp: Response{
				Text:    "this is spam from \"testuser\", but I'm in dry mode, so I'll do nothing yet",
				Send:    true,
				ReplyTo: 1,
			},
		},
		{
			name:           "HTTP error",
			mockResp:       "",
			mockStatusCode: 500,
			dryMode:        false,
			expectedResp:   Response{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockedHTTPClient := &mocks.HTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tt.mockStatusCode,
						Body:       io.NopCloser(bytes.NewBufferString(tt.mockResp)),
					}, nil
				},
			}

			s := NewSpamFilter("http://localhost", mockedHTTPClient, tt.dryMode)

			msg := Message{
				From: User{
					ID:          1,
					Username:    "testuser",
					DisplayName: "Test User",
				},
				ID:   1,
				Text: "Hello",
			}

			resp := s.OnMessage(msg)
			assert.Equal(t, tt.expectedResp, resp)
		})
	}
}

func TestSpamFilter_OnMessageCheckOnce(t *testing.T) {
	mockedHTTPClient := &mocks.HTTPClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(bytes.NewBufferString(`{"ok": false, "description": "Not a spammer"}`)),
			}, nil
		},
	}

	s := NewSpamFilter("http://localhost", mockedHTTPClient, false)
	res := s.OnMessage(Message{From: User{ID: 1, Username: "testuser"}, ID: 1, Text: "Hello"})
	assert.Equal(t, Response{}, res)
	assert.Len(t, mockedHTTPClient.DoCalls(), 1, "Do should be called once")

	res = s.OnMessage(Message{From: User{ID: 1, Username: "testuser"}, ID: 1, Text: "Hello"})
	assert.Equal(t, Response{}, res)
	assert.Len(t, mockedHTTPClient.DoCalls(), 1, "Do should not be called anymore")

	res = s.OnMessage(Message{From: User{ID: 2, Username: "testuser2"}, ID: 2, Text: "Hello"})
	assert.Equal(t, Response{}, res)
	assert.Len(t, mockedHTTPClient.DoCalls(), 2, "Do should be called once more")
}
