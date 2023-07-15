package events

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/go-pkgz/syncs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/radio-t/super-bot/app/events/mocks"
)

func TestRtjc_isPinned(t *testing.T) {
	tbl := []struct {
		inp string
		out string
		pin bool
	}{
		{"blah", "blah", false},
		{"⚠️ Официальный кАт! - https://stream.radio-t.com/", "⚠️ Вещание подкаста началось - https://stream.radio-t.com/", true},
		{" ⚠️ Официальный кАт! - https://stream.radio-t.com/ ", "⚠️ Вещание подкаста началось - https://stream.radio-t.com/", true},
		{" ⚠️ Официальный кАт! - https://stream.radio-t.com/\n", "⚠️ Вещание подкаста началось - https://stream.radio-t.com/", true},
	}

	rtjc := Rtjc{}
	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			pin, out := rtjc.isPinned(tt.inp)
			assert.Equal(t, tt.pin, pin)
			assert.Equal(t, tt.out, out)
		})
	}
}

func makeTestingRtjc(submitter *mocks.Submitter, summarizer *mocks.Summarizer) Rtjc {
	return Rtjc{
		Port:            1,
		Submitter:       submitter,
		Summarizer:      summarizer,
		Swg:             syncs.NewSizedGroup(1),
		SubmitRateLimit: 1,
		SubmitRateBurst: 100,
	}
}

func TestRtjc_ReadMessage(t *testing.T) {
	tbl := []struct {
		name            string
		input           string
		callsSubmit     int
		callsSubmitHTML int
		callsSummary    int
	}{
		{"Begin", "⚠️ Вещание подкаста началось - https://stream.radio-t.com/", 1, 0, 1},
		{"New theme", "⚠️ blah blah - https://link.example.com", 1, 1, 1},
		{"Remark", "⚠️ blah blah - https://radio-t.com/p/2023/04/04/prep-853/", 1, 2, 1},
		{"Blah", "blah blah - https://link.example.com", 1, 0, 0},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			sb := &mocks.Submitter{
				SubmitFunc: func(ctx context.Context, text string, pin bool) error {
					return nil
				},
				SubmitHTMLFunc: func(ctx context.Context, text string, pin bool) error {
					return nil
				},
			}

			sm := &mocks.Summarizer{
				GetSummariesByMessageFunc: func(link string) (messages []string, err error) {
					// Begin and end of stream
					if strings.Contains(link, "stream.radio-t.com") {
						return []string{}, nil
					}

					// Remark link
					if strings.Contains(link, "radio-t.com/p/") {
						return []string{"summary", "summary2"}, nil
					}

					// Regular link
					return []string{"summary"}, nil
				},
			}

			var buf bytes.Buffer
			buf.WriteString(tt.input + "\n")

			rtjc := makeTestingRtjc(sb, sm)
			rtjc.processMessage(context.Background(), &buf)
			rtjc.Swg.Wait()

			require.Equal(t, tt.callsSubmit, len(sb.SubmitCalls()))
			// If number of calls is 1, then this is regular message with link (non-remark)
			// so we should check the content of Submit call
			if tt.callsSubmit == 1 {
				assert.Contains(t, sb.SubmitCalls()[0].Text, tt.input)
			}
			assert.Equal(t, tt.callsSubmitHTML, len(sb.SubmitHTMLCalls()))
			assert.Equal(t, tt.callsSummary, len(sm.GetSummariesByMessageCalls()))
		})
	}
}

func TestRtjc_SendSummary(t *testing.T) {
	tbl := []struct {
		name            string
		input           string
		summaryResponse []string
		callsSubmit     int
		callsSubmitHTML int
		callsSummary    int
	}{
		{"Begin", "⚠️ Вещание подкаста началось - https://stream.radio-t.com/", []string{}, 0, 0, 1},
		{"New theme", "⚠️ blah blah - https://link.example.com", []string{"summary"}, 0, 1, 1},
		{"Remark", "⚠️ blah blah - https://radio-t.com/p/2023/04/04/prep-853/", []string{"summary1/2", "summary2/2"}, 0, 2, 1},
		{"Remark with empty summaries", "⚠️ blah blah - https://radio-t.com/p/2023/04/04/prep-853/", []string{"summary1/3", "", "summary2/3", "summary3/3"}, 0, 3, 1},
		{"Blah", "blah blah - https://link.example.com", []string{}, 0, 0, 0},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			sb := &mocks.Submitter{
				SubmitFunc: func(ctx context.Context, text string, pin bool) error {
					return nil
				},
				SubmitHTMLFunc: func(ctx context.Context, text string, pin bool) error {
					return nil
				},
			}

			sm := &mocks.Summarizer{
				GetSummariesByMessageFunc: func(link string) (messages []string, err error) {
					return tt.summaryResponse, nil
				},
			}

			rtjc := makeTestingRtjc(sb, sm)
			rtjc.sendSummary(context.Background(), tt.input+"\n")
			rtjc.Swg.Wait()

			assert.Equal(t, tt.callsSubmit, len(sb.SubmitCalls()))
			require.Equal(t, tt.callsSubmitHTML, len(sb.SubmitHTMLCalls()))

			expectedCalls := make([]string, 0)
			for _, re := range tt.summaryResponse {
				if re != "" {
					expectedCalls = append(expectedCalls, re)
				}
			}
			actualCalls := make([]string, 0)
			for _, call := range sb.SubmitHTMLCalls() {
				actualCalls = append(actualCalls, call.Text)
			}
			assert.ElementsMatch(t, expectedCalls, actualCalls)

			assert.Equal(t, tt.callsSummary, len(sm.GetSummariesByMessageCalls()))
		})
	}
}
