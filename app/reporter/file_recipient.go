package reporter

import (
	"io"
	"net/http"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
)

// FileRecipient knows how to get file by fileID
// Needed mainly for tests to mock
type FileRecipient interface {
	GetFile(fileID string) (io.ReadCloser, error)
}

// TelegramFileRecipient implements FileRecipient for Telegram files
type TelegramFileRecipient struct {
	botAPI     *tbapi.BotAPI
	httpClient *http.Client
}

// NewTelegramFileRecipient creates TelegramFileRecipient
func NewTelegramFileRecipient(botAPI *tbapi.BotAPI, timeout time.Duration) FileRecipient {
	return &TelegramFileRecipient{
		botAPI: botAPI,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetFile gets file from Telegram
func (tfd TelegramFileRecipient) GetFile(fileID string) (io.ReadCloser, error) {
	url, err := tfd.botAPI.GetFileDirectURL(fileID)
	if err != nil {
		return nil, errors.Wrapf(err, "get file direct URL (fileID: %s)", fileID)
	}

	resp, err := tfd.httpClient.Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "file by direct URL (fileID: %s)", fileID)
		// don't expose `url` – it contains Bot API Token
	}

	if resp.StatusCode != 200 {
		return nil, errors.Wrapf(err, "non-200 response from file direct URL (fileID: %s)", fileID)
	}

	return resp.Body, nil
}
