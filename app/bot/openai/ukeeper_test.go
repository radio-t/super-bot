package openai

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUKeeperClient_Get(t *testing.T) {
	tbl := []struct {
		name            string
		response        string
		expectedTitle   string
		expectedContent string
		err             string
	}{
		{"Good content-type: text/html", `{"Content": "some Content", "Title": "some Title", "Type": "text/html"}`, "some Title", "some Content", ""},
		{"Bad content-type: image/jpeg", `{"Content": "some Content", "Title": "some Title", "Type": "image/jpeg"}`, "", "", "bad content type http://radio-t.com: image/jpeg"},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "token123", r.URL.Query().Get("token"))
				_, err := w.Write([]byte(tt.response))
				require.NoError(t, err)
			}))

			uk := UKeeperClient{
				API:    ts.URL,
				Token:  "token123",
				Client: ts.Client(),
			}

			title, content, err := uk.Get("http://radio-t.com")
			if tt.err == "" {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tt.err, err.Error())
			}
			assert.Equal(t, tt.expectedTitle, title)
			assert.Equal(t, tt.expectedContent, content)
		})
	}
}
