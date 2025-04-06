package openai

import (
	"fmt"
	"testing"
	"time"

	"github.com/go-pkgz/lcw/v2"
	"github.com/radio-t/super-bot/app/bot/openai/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func genTestRemarkComment(user, text string, score int) remarkComment {
	return remarkComment{
		ParentID: "0",
		Text:     text,
		User: struct {
			Name     string `json:"name"`
			Admin    bool   `json:"admin"`
			Verified bool   `json:"verified,omitempty"`
		}{Name: user, Admin: false, Verified: true},
		Score: score,
	}
}

func TestSummarizer_GetSummariesByMessage(t *testing.T) {
	uk := &mocks.UKeeperClient{
		GetFunc: func(link string) (title, content string, err error) {
			return "Title ABC", "Content " + link, nil
		},
	}

	rc := &mocks.RemarkClient{
		GetTopCommentsFunc: func(remarkLink string) (comments []string, links []string, err error) {
			return []string{}, []string{}, nil
		},
	}

	os := &mocks.OpenAISummary{
		SummaryFunc: func(text string) (summary string, err error) {
			return "Summary: " + text, nil
		},
	}

	o := lcw.NewOpts[summaryItem]()
	cache, err := lcw.NewExpirableCache(o.MaxKeys(10), o.TTL(time.Hour))
	assert.NoError(t, err)
	s := Summarizer{
		openAISummary: os,
		remark:        rc,
		uKeeper:       uk,
		cache:         cache,
		threads:       1,
		debug:         false,
	}

	summaries, err := s.GetSummariesByMessage("some message blah https://example.theme.com")
	assert.NoError(t, err)

	require.Equal(t, 1, len(summaries))
	assert.Equal(t, "<b>Title ABC</b>\n\nSummary: Title ABC - Content https://example.theme.com", summaries[0])

	assert.Equal(t, 1, len(os.SummaryCalls()))
	assert.Equal(t, 0, len(rc.GetTopCommentsCalls()))
	assert.Equal(t, 1, len(uk.GetCalls()))
}

func TestSummarizer_GetSummariesByMessageRemark(t *testing.T) {
	uk := &mocks.UKeeperClient{
		GetFunc: func(link string) (title, content string, err error) {
			return "Title ABC", "Content " + link, nil
		},
	}

	rc := &mocks.RemarkClient{
		GetTopCommentsFunc: func(remarkLink string) (comments []string, links []string, err error) {
			c1 := genTestRemarkComment("User1", "some message blah <a href=\"https://example.user1.com\">Link</a>", 2)
			c2 := genTestRemarkComment("User2", "some message blah <a href=\"https://example.user2.com\">Link</a>", 1)
			return []string{c1.render(), c2.render()}, []string{c1.getLink(), c2.getLink()}, nil
		},
	}

	os := &mocks.OpenAISummary{
		SummaryFunc: func(text string) (summary string, err error) {
			return "Summary: " + text, nil
		},
	}

	o := lcw.NewOpts[summaryItem]()
	cache, err := lcw.NewExpirableCache(o.MaxKeys(10), o.TTL(time.Hour))
	assert.NoError(t, err)
	s := Summarizer{
		openAISummary: os,
		remark:        rc,
		uKeeper:       uk,
		cache:         cache,
		threads:       1,
		debug:         false,
	}

	summaries, err := s.GetSummariesByMessage("some message blah https://radio-t.com/p/2023/04/04/prep-853/")
	assert.NoError(t, err)

	require.Equal(t, 2, len(summaries))
	assert.Equal(t, "[1/2] <b>+2</b> от <b>User1</b>\n<i>some message blah <a href=\"https://example.user1.com\">Link</a></i>\n\n<b>Title ABC</b>\n\nSummary: Title ABC - Content https://example.user1.com", summaries[0])
	assert.Equal(t, "[2/2] <b>+1</b> от <b>User2</b>\n<i>some message blah <a href=\"https://example.user2.com\">Link</a></i>\n\n<b>Title ABC</b>\n\nSummary: Title ABC - Content https://example.user2.com", summaries[1])

	assert.Equal(t, 2, len(os.SummaryCalls()))
	assert.Equal(t, 1, len(rc.GetTopCommentsCalls()))
	assert.Equal(t, 2, len(uk.GetCalls()))
}

func TestSummarizer_GetSummariesByMessage_ErrorNoLink(t *testing.T) {
	uk := &mocks.UKeeperClient{
		GetFunc: func(link string) (title, content string, err error) {
			return "Title ABC", "Content " + link, nil
		},
	}

	rc := &mocks.RemarkClient{
		GetTopCommentsFunc: func(remarkLink string) (comments []string, links []string, err error) {
			return []string{}, []string{}, nil
		},
	}

	os := &mocks.OpenAISummary{
		SummaryFunc: func(text string) (summary string, err error) {
			return "Summary: " + text, nil
		},
	}

	o := lcw.NewOpts[summaryItem]()
	cache, err := lcw.NewExpirableCache(o.MaxKeys(10), o.TTL(time.Hour))
	assert.NoError(t, err)
	s := Summarizer{
		openAISummary: os,
		remark:        rc,
		uKeeper:       uk,
		cache:         cache,
		threads:       1,
		debug:         false,
	}

	_, err = s.GetSummariesByMessage("some message blah")
	assert.Error(t, err)

	assert.Equal(t, 0, len(os.SummaryCalls()))
	assert.Equal(t, 0, len(rc.GetTopCommentsCalls()))
	assert.Equal(t, 0, len(uk.GetCalls()))
}

func TestSummarizer_GetSummariesByMessage_ErrorSummary(t *testing.T) {
	uk := &mocks.UKeeperClient{
		GetFunc: func(link string) (title, content string, err error) {
			return "Title ABC", "Content " + link, nil
		},
	}

	rc := &mocks.RemarkClient{
		GetTopCommentsFunc: func(remarkLink string) (comments []string, links []string, err error) {
			return []string{}, []string{}, nil
		},
	}

	os := &mocks.OpenAISummary{
		SummaryFunc: func(text string) (summary string, err error) {
			return "", fmt.Errorf("some error")
		},
	}

	o := lcw.NewOpts[summaryItem]()
	cache, err := lcw.NewExpirableCache(o.MaxKeys(10), o.TTL(time.Hour))
	assert.NoError(t, err)
	s := Summarizer{
		openAISummary: os,
		remark:        rc,
		uKeeper:       uk,
		cache:         cache,
		threads:       1,
		debug:         false,
	}

	_, err = s.GetSummariesByMessage("some message blah https://example.theme.com")
	assert.Error(t, err)

	assert.Equal(t, 1, len(os.SummaryCalls()))
	assert.Equal(t, 0, len(rc.GetTopCommentsCalls()))
	assert.Equal(t, 1, len(uk.GetCalls()))
}

func TestSummarizer_GetSummariesByMessage_ErrorBadRadiotLink(t *testing.T) {
	uk := &mocks.UKeeperClient{
		GetFunc: func(link string) (title, content string, err error) {
			return "Title ABC", "Content " + link, nil
		},
	}

	rc := &mocks.RemarkClient{
		GetTopCommentsFunc: func(remarkLink string) (comments []string, links []string, err error) {
			return []string{}, []string{}, nil
		},
	}

	os := &mocks.OpenAISummary{
		SummaryFunc: func(text string) (summary string, err error) {
			return "", fmt.Errorf("some error")
		},
	}

	o := lcw.NewOpts[summaryItem]()
	cache, err := lcw.NewExpirableCache(o.MaxKeys(10), o.TTL(time.Hour))
	assert.NoError(t, err)
	s := Summarizer{
		openAISummary: os,
		remark:        rc,
		uKeeper:       uk,
		cache:         cache,
		threads:       1,
		debug:         false,
	}

	_, err = s.GetSummariesByMessage("some message blah https://radio-t.com/about/")
	assert.Error(t, err)

	assert.Equal(t, 0, len(os.SummaryCalls()))
	assert.Equal(t, 0, len(rc.GetTopCommentsCalls()))
	assert.Equal(t, 0, len(uk.GetCalls()))
}

func TestSummarizer_GetSummariesByMessage_ErrorGetComments(t *testing.T) {
	uk := &mocks.UKeeperClient{
		GetFunc: func(link string) (title, content string, err error) {
			return "Title ABC", "Content " + link, nil
		},
	}

	rc := &mocks.RemarkClient{
		GetTopCommentsFunc: func(remarkLink string) (comments []string, links []string, err error) {
			return []string{}, []string{}, fmt.Errorf("some error")
		},
	}

	os := &mocks.OpenAISummary{
		SummaryFunc: func(text string) (summary string, err error) {
			return "", fmt.Errorf("some error")
		},
	}

	o := lcw.NewOpts[summaryItem]()
	cache, err := lcw.NewExpirableCache(o.MaxKeys(10), o.TTL(time.Hour))
	assert.NoError(t, err)
	s := Summarizer{
		openAISummary: os,
		remark:        rc,
		uKeeper:       uk,
		cache:         cache,
		threads:       1,
		debug:         false,
	}

	_, err = s.GetSummariesByMessage("some message blah https://radio-t.com/p/2023/04/04/prep-853/")
	assert.Error(t, err)

	assert.Equal(t, 0, len(os.SummaryCalls()))
	assert.Equal(t, 1, len(rc.GetTopCommentsCalls()))
	assert.Equal(t, 0, len(uk.GetCalls()))
}

func TestSummarizer_GetSummariesByMessage_ErrorGetContent(t *testing.T) {
	uk := &mocks.UKeeperClient{
		GetFunc: func(link string) (title, content string, err error) {
			return "", "", fmt.Errorf("some error")
		},
	}

	rc := &mocks.RemarkClient{
		GetTopCommentsFunc: func(remarkLink string) (comments []string, links []string, err error) {
			return []string{}, []string{}, fmt.Errorf("some error")
		},
	}

	os := &mocks.OpenAISummary{
		SummaryFunc: func(text string) (summary string, err error) {
			return "", nil
		},
	}

	o := lcw.NewOpts[summaryItem]()
	cache, err := lcw.NewExpirableCache(o.MaxKeys(10), o.TTL(time.Hour))
	assert.NoError(t, err)
	s := Summarizer{
		openAISummary: os,
		remark:        rc,
		uKeeper:       uk,
		cache:         cache,
		threads:       1,
		debug:         false,
	}

	_, err = s.GetSummariesByMessage("some message blah https://example.theme.com")
	assert.Error(t, err)

	assert.Equal(t, 0, len(os.SummaryCalls()))
	assert.Equal(t, 0, len(rc.GetTopCommentsCalls()))
	assert.Equal(t, 1, len(uk.GetCalls()))
}

func TestSummarizer_GetSummariesByRemarkLink(t *testing.T) {
	uk := &mocks.UKeeperClient{
		GetFunc: func(link string) (title, content string, err error) {
			return "Title ABC", "Content " + link, nil
		},
	}

	rc := &mocks.RemarkClient{
		GetTopCommentsFunc: func(remarkLink string) (comments []string, links []string, err error) {
			c1 := genTestRemarkComment("User1", "some message blah <a href=\"https://example.user1.com\">Link</a>", 2)
			c2 := genTestRemarkComment("User2", "some message blah <a href=\"https://example.user2.com\">Link</a>", 1)
			return []string{c1.render(), c2.render()}, []string{c1.getLink(), c2.getLink()}, nil
		},
	}

	os := &mocks.OpenAISummary{
		SummaryFunc: func(text string) (summary string, err error) {
			return "Summary: " + text, nil
		},
	}

	o := lcw.NewOpts[summaryItem]()
	cache, err := lcw.NewExpirableCache(o.MaxKeys(10), o.TTL(time.Hour))
	assert.NoError(t, err)
	s := Summarizer{
		openAISummary: os,
		remark:        rc,
		uKeeper:       uk,
		cache:         cache,
		threads:       1,
		debug:         false,
	}

	summaries, err := s.GetSummariesByRemarkLink("https://radio-t.com/p/2023/04/04/prep-853/")
	assert.NoError(t, err)

	require.Equal(t, 2, len(summaries))
	assert.Equal(t, "[1/2] <b>+2</b> от <b>User1</b>\n<i>some message blah <a href=\"https://example.user1.com\">Link</a></i>\n\n<b>Title ABC</b>\n\nSummary: Title ABC - Content https://example.user1.com", summaries[0])
	assert.Equal(t, "[2/2] <b>+1</b> от <b>User2</b>\n<i>some message blah <a href=\"https://example.user2.com\">Link</a></i>\n\n<b>Title ABC</b>\n\nSummary: Title ABC - Content https://example.user2.com", summaries[1])

	require.Equal(t, 2, len(uk.GetCalls()))
	// parallel execution may change order of calls
	expected := []string{
		"https://example.user1.com",
		"https://example.user2.com",
	}
	actual := make([]string, 0)
	for _, call := range uk.GetCalls() {
		actual = append(actual, call.Link)
	}
	assert.ElementsMatch(t, expected, actual)

	require.Equal(t, 2, len(os.SummaryCalls()))
	// parallel execution may change order of calls
	expected = []string{
		"Title ABC - Content https://example.user1.com",
		"Title ABC - Content https://example.user2.com",
	}
	actual = make([]string, 0)
	for _, call := range os.SummaryCalls() {
		actual = append(actual, call.Text)
	}
	assert.ElementsMatch(t, expected, actual)

	assert.Equal(t, 1, len(rc.GetTopCommentsCalls()))
	assert.Equal(t, "https://radio-t.com/p/2023/04/04/prep-853/", rc.GetTopCommentsCalls()[0].RemarkLink)
}

func TestSummarizer_Summary(t *testing.T) {
	uc := &mocks.UKeeperClient{
		GetFunc: func(link string) (title, content string, err error) {
			return "Title ABC", "Content CBA", nil
		},
	}

	os := &mocks.OpenAISummary{
		SummaryFunc: func(link string) (content string, err error) {
			return "Summary ABC", nil
		},
	}

	o := lcw.NewOpts[summaryItem]()
	cache, err := lcw.NewExpirableCache(o.MaxKeys(10), o.TTL(time.Hour))
	assert.NoError(t, err)
	s := Summarizer{
		openAISummary: os,
		uKeeper:       uc,
		cache:         cache,
		debug:         false,
	}

	summary, err := s.Summary("https://radio-t.com")
	assert.NoError(t, err)
	assert.Equal(t, "<b>Title ABC</b>\n\nSummary ABC", summary)

	require.Equal(t, 1, len(uc.GetCalls()))
	assert.Equal(t, "https://radio-t.com", uc.GetCalls()[0].Link)
	require.Equal(t, 1, len(os.SummaryCalls()))
	assert.Equal(t, "Title ABC - Content CBA", os.SummaryCalls()[0].Text)
}

func TestSummaryItem_Render(t *testing.T) {
	tbl := []struct {
		name     string
		title    string
		content  string
		expected string
	}{
		{"Good", "Title ABC", "Content CBA", "<b>Title ABC</b>\n\nContent CBA"},
		{"Bad: empty title", "", "Content CBA", ""},
		{"Bad: empty content", "Title ABC", "", ""},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			si := summaryItem{
				Title:   tt.title,
				Content: tt.content,
			}

			assert.Equal(t, tt.expected, si.render())
		})
	}
}
