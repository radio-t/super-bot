package reporter

import (
	"fmt"
	"io"
	"net/http"
	"time"

	tbapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
		return nil, fmt.Errorf("get file direct URL (fileID: %s): %w", fileID, err)
	}

	resp, err := tfd.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("file by direct URL (fileID: %s): %w", fileID, err)
		// don't expose `url` – it contains Bot API Token
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("non-200 response from file direct URL (fileID: %s): %w", fileID, err)
	}

	return resp.Body, nil
}
