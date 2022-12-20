package bot

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/radio-t/super-bot/app/bot/mocks"
)

func TestPrepPost_OnMessage(t *testing.T) {
	tbl := []struct {
		body   string
		err    error
		status int
		resp   Response
	}{
		{
			`[{"url":"https://radio-t.com/p/2020/02/11/prep-689/","title":"Темы для 689","date":"2020-02-11T23:04:21Z","categories":["prep"]}]`,
			nil, 200, Response{},
		},
		{
			`[{"url":"https://radio-t.com/p/2020/02/11/prep-689/","title":"Темы для 689","date":"2020-02-11T23:04:21Z","categories":["prep"]}]`,
			nil, 200, Response{},
		},
		{
			`[{"url":"https://radio-t.com/p/2020/02/11/prep-689/","title":"Темы для 689","date":"2020-02-11T23:04:21Z","categories":["prep"]}]`,
			nil, 200, Response{},
		},
		{
			"errrrr", nil, 400, Response{},
		},
		{
			"", fmt.Errorf("error"), 200, Response{},
		},
		{
			`[{"url":"https://radio-t.com/p/2020/02/11/prep-690/","title":"Темы для 690","date":"2020-02-11T23:04:21Z","categories":["prep"]}]`,
			nil, 200, Response{Text: "Сбор тем начался - https://radio-t.com/p/2020/02/11/prep-690/", Send: true, Pin: true, Preview: false},
		},
		{
			`[{"url":"https://radio-t.com/p/2020/02/11/prep-690/","title":"Темы для 690","date":"2020-02-11T23:04:21Z","categories":["prep"]}]`,
			nil, 200, Response{},
		},
	}

	mockHTTP := &mocks.HTTPClient{}
	pp := NewPrepPost(mockHTTP, "http://example.com", time.Millisecond*10)

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			mockHTTP.DoFunc = func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					Body:       io.NopCloser(strings.NewReader(tt.body)),
					StatusCode: tt.status,
				}, tt.err
			}
			resp := pp.OnMessage(Message{})
			assert.Equal(t, tt.resp, resp)
			time.Sleep(time.Millisecond * 11)
		})
	}

}

func TestPrepPost_checkDuration(t *testing.T) {
	hit := false
	mockHTTP := &mocks.HTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
		if !hit {
			hit = true
			return &http.Response{
				Body:       io.NopCloser(strings.NewReader(`[{"url":"blah1","title":"Темы для 689","categories":["prep"]}]`)),
				StatusCode: 200,
			}, nil
		}
		return &http.Response{
			Body:       io.NopCloser(strings.NewReader(`[{"url":"blah2","title":"Темы для 689","categories":["prep"]}]`)),
			StatusCode: 200,
		}, nil
	}}
	pp := NewPrepPost(mockHTTP, "http://example.com", time.Millisecond*50)

	for i := 0; i < 10; i++ {
		pp.OnMessage(Message{})
		time.Sleep(6 * time.Millisecond)
	}

	assert.Equal(t, 2, len(mockHTTP.DoCalls()))
}
