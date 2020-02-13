package bot

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
			"", errors.New("error"), 200, Response{},
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

	mockHTTP := &MockHTTPClient{}
	pp := NewPrepPost(mockHTTP, "http://example.com", time.Millisecond*10)

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			mockHTTP.On("Do", mock.Anything).Return(&http.Response{
				Body:       ioutil.NopCloser(strings.NewReader(tt.body)),
				StatusCode: tt.status,
			}, tt.err).Times(1)
			resp := pp.OnMessage(Message{})
			assert.Equal(t, tt.resp, resp)
			time.Sleep(time.Millisecond * 11)
		})
	}

}

func TestPrepPost_checkDuration(t *testing.T) {
	mockHTTP := &MockHTTPClient{}
	pp := NewPrepPost(mockHTTP, "http://example.com", time.Millisecond*50)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body:       ioutil.NopCloser(strings.NewReader(`[{"url":"blah1","title":"Темы для 689","categories":["prep"]}]`)),
		StatusCode: 200,
	}, nil).Times(1)

	mockHTTP.On("Do", mock.Anything).Return(&http.Response{
		Body:       ioutil.NopCloser(strings.NewReader(`[{"url":"blah2","title":"Темы для 689","categories":["prep"]}]`)),
		StatusCode: 200,
	}, nil).Times(1)

	for i := 0; i < 10; i++ {
		pp.OnMessage(Message{})
		time.Sleep(6 * time.Millisecond)
	}

	mockHTTP.AssertNumberOfCalls(t, "Do", 2)
	mockHTTP.AssertExpectations(t)
}
