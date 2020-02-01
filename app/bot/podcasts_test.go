package bot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodcasts_OnMessage(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/search" {
			assert.Equal(t, "limit=5&q=something", r.URL.RawQuery)
			sr := []siteAPIResp{
				{
					URL:       "http://example.com",
					Date:      time.Date(2020, 1, 31, 16, 45, 0, 0, time.UTC),
					ShowNum:   123,
					ShowNotes: "\n\n\nGo 2 начинается - 00:01:31.\nAWS Transfer for SFTP - 00:19:39.\nAWS App Mesh - 00:33:39.\nAmazon DynamoDB On-Demand - 00:46:50.\nALB сможет вызвать Lambda - 00:54:45.\nСлои общего кода в AWS Lambda - 01:15:46.\nDrone Cloud и бесплатно - 01:21:39.\nFoundationDB Document Layer совместим с mongo - 01:41:31.\nТемы наших слушателей\n\n\nСпонсор этого выпуска DigitalOcean\n\nаудио • лог чата\n\n",
				},
			}
			b, err := json.Marshal(sr)
			require.NoError(t, err)
			w.Write(b)
			return
		}
		w.WriteHeader(400)
	}))
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	d := NewPodcasts(&client, ts.URL, 5)

	result, answer := d.OnMessage(Message{Text: "/search something"})
	require.True(t, answer)
	assert.Equal(t, "[Радио-Т #123](http://example.com) _31 Jan 20_\n\n- Go 2 начинается - 00:01:31.\n- AWS Transfer for SFTP - 00:19:39.\n- AWS App Mesh - 00:33:39.\n- Amazon DynamoDB On-Demand - 00:46:50.\n- ALB сможет вызвать Lambda - 00:54:45.\n- Слои общего кода в AWS Lambda - 01:15:46.\n- Drone Cloud и бесплатно - 01:21:39.\n- FoundationDB Document Layer совместим с mongo - 01:41:31.\n- Темы наших слушателей\n- Спонсор этого выпуска DigitalOcean\n\n", result)

	_, answer = d.OnMessage(Message{Text: "/search something"})
	require.True(t, answer)
}

func TestPodcasts_OnMessageIgnore(t *testing.T) {

	d := NewPodcasts(&http.Client{}, "http://example.com", 5)

	_, answer := d.OnMessage(Message{Text: "/xyz something"})
	require.False(t, answer)
}

func TestPodcasts_OnMessageFailed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer ts.Close()

	client := http.Client{Timeout: time.Second}
	d := NewPodcasts(&client, ts.URL, 5)

	_, answer := d.OnMessage(Message{Text: "/search something"})
	require.False(t, answer)
}
