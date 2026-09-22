package jev

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testQuestions() map[string]Question {
	return map[string]Question{
		"invites": {Type: "noul", Instructions: "Does the message invite an answer?", Criteria: map[string]string{
			"true": "A genuine request", "false": "A statement",
		}},
	}
}

func TestClient_Ask(t *testing.T) {
	questions := testQuestions()
	questions["answerable"] = Question{Type: "noul", Instructions: "Is the context sufficient?"}
	questions["spam"] = Question{Type: "noul", Instructions: "Is this unsolicited advertising?"}
	state := map[string]string{"message": "What does a compiler do?"}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/systemone", r.URL.Path)
		assert.Equal(t, "Bearer fake-jev-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var payload struct {
			Model     string              `json:"model"`
			State     map[string]string   `json:"state"`
			Questions map[string]Question `json:"questions"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		assert.Equal(t, "jev-1.13.0", payload.Model)
		assert.Equal(t, state, payload.State)
		assert.Equal(t, questions, payload.Questions)
		_, _ = io.WriteString(w, `{"model":"jev-resolved","answers":{"invites":{"type":"noul","noul":1},"answerable":{"type":"noul","noul":0.75},"spam":{"type":"noul","noul":0}},"usage":{"input_tokens":10}}`)
	}))
	t.Cleanup(server.Close)
	client := New(Params{Key: "fake-jev-key", Model: "jev-1.13.0", Timeout: time.Second})
	client.endpoint = server.URL + "/v1/systemone"

	response, err := client.Ask(context.Background(), state, questions)
	require.NoError(t, err)
	assert.Equal(t, "jev-resolved", response.Model)
	require.Len(t, response.Answers, 3)
	for key, want := range map[string]float64{"invites": 1, "answerable": 0.75, "spam": 0} {
		answer := response.Answers[key]
		require.NotNil(t, answer.Noul)
		assert.Equal(t, "noul", answer.Type)
		assert.InDelta(t, want, *answer.Noul, 0.00001)
	}
	assert.EqualValues(t, 1, calls.Load())
}

func TestClient_Ask_InvalidResponse(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"missing answer", `{"model":"jev","answers":{}}`, "missing answer"},
		{"null answer", `{"model":"jev","answers":{"invites":null}}`, "answer type"},
		{"wrong type", `{"model":"jev","answers":{"invites":{"type":"choice","noul":0.8}}}`, "answer type"},
		{"missing type", `{"model":"jev","answers":{"invites":{"noul":0.8}}}`, "answer type"},
		{"missing noul", `{"model":"jev","answers":{"invites":{"type":"noul"}}}`, "noul"},
		{"null noul", `{"model":"jev","answers":{"invites":{"type":"noul","noul":null}}}`, "noul"},
		{"negative noul", `{"model":"jev","answers":{"invites":{"type":"noul","noul":-0.01}}}`, "noul"},
		{"noul above one", `{"model":"jev","answers":{"invites":{"type":"noul","noul":1.01}}}`, "noul"},
		{"string noul", `{"model":"jev","answers":{"invites":{"type":"noul","noul":"yes"}}}`, "decode"},
		{"overflow noul", `{"model":"jev","answers":{"invites":{"type":"noul","noul":1e999}}}`, "decode"},
		{"missing model", `{"answers":{"invites":{"type":"noul","noul":0.8}}}`, "model"},
		{"malformed JSON", `{"model":`, "decode"},
		{"trailing JSON", `{"model":"jev","answers":{"invites":{"type":"noul","noul":0.8}}}{}`, "decode"},
		{"oversized body", strings.Repeat(" ", 65537), "too large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, tt.body)
			}))
			t.Cleanup(server.Close)
			client := New(Params{Key: "fake-jev-key", Model: "jev", Timeout: time.Second})
			client.endpoint = server.URL

			response, err := client.Ask(context.Background(), "message", testQuestions())
			require.ErrorContains(t, err, tt.want)
			assert.Empty(t, response)
		})
	}
}

func TestClient_Ask_HTTPError(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"unknown model", 400, `{"detail":{"error_type":"api_usage_error","message":"Unknown model: jev-nope"}}`, "api_usage_error: Unknown model: jev-nope"},
		{"bad key", 401, `{"detail":{"error_type":"authentication_error","message":"Cannot authenticate..."}}`, "authentication_error: Cannot authenticate..."},
		{"rate limited", 429, `{"detail":{"error_type":"rate_limit","message":"Try later"}}`, "rate_limit: Try later"},
		{"overloaded", 529, `{"detail":{"error_type":"overloaded","message":"Unavailable"}}`, "overloaded: Unavailable"},
		{"validation", 422, `{"detail":[{"type":"missing","loc":["body","state"],"msg":"Field required","input":{"text":"private-state-marker","key":"fake-jev-key"},"ctx":{"secret":"private-state-marker"}}]}`, "missing: Field required"},
		{"key in message", 401, `{"detail":{"error_type":"authentication_error","message":"Invalid fake-jev-key"}}`, "authentication_error: Invalid [redacted]"},
		{"string detail", 503, `{"detail":"temporarily\n  unavailable fake-jev-key"}`, "temporarily unavailable [redacted]"},
		{"unstructured body", 502, "private-state-marker fake-jev-key", "HTTP 502"},
		{"no safe fields", 422, `{"input":"private-state-marker","key":"fake-jev-key"}`, "HTTP 422"},
		{"long message", 500, `{"detail":{"error_type":"internal","message":"` + strings.Repeat("x", 500) + `"}}`, "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			t.Cleanup(server.Close)
			client := New(Params{Key: "fake-jev-key", Model: "jev", Timeout: time.Second})
			client.endpoint = server.URL

			response, err := client.Ask(context.Background(), "private-state-marker", testQuestions())
			require.ErrorContains(t, err, fmt.Sprintf("HTTP %d", tt.status))
			assert.ErrorContains(t, err, tt.want)
			assert.NotContains(t, err.Error(), "private-state-marker")
			assert.NotContains(t, err.Error(), "fake-jev-key")
			assert.LessOrEqual(t, len(err.Error()), 300)
			assert.Empty(t, response)
			assert.EqualValues(t, 1, calls.Load())
		})
	}
}

func TestClient_Ask_Timeout(t *testing.T) {
	for _, bodyStarted := range []bool{false, true} {
		t.Run(fmt.Sprintf("body_started_%t", bodyStarted), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if bodyStarted {
					w.WriteHeader(http.StatusOK)
					w.(http.Flusher).Flush()
				}
				select {
				case <-r.Context().Done():
				case <-time.After(time.Second):
				}
			}))
			t.Cleanup(server.Close)
			client := New(Params{Key: "fake-jev-key", Model: "jev", Timeout: 20 * time.Millisecond})
			client.endpoint = server.URL

			response, err := client.Ask(context.Background(), "message", testQuestions())
			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Empty(t, response)
		})
	}
}

func TestClient_Ask_Redirect(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Redirect(w, r, "/redirected", http.StatusTemporaryRedirect)
	}))
	t.Cleanup(server.Close)
	client := New(Params{Key: "fake-jev-key", Model: "jev", Timeout: time.Second})
	client.endpoint = server.URL

	_, err := client.Ask(context.Background(), "message", testQuestions())
	require.ErrorContains(t, err, "HTTP 307")
	assert.EqualValues(t, 1, calls.Load())
}

func TestClient_Ask_ReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "{")
	}))
	t.Cleanup(server.Close)
	client := New(Params{Key: "fake-jev-key", Model: "jev", Timeout: time.Second})
	client.endpoint = server.URL

	_, err := client.Ask(context.Background(), "message", testQuestions())
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestClient_Ask_InvalidRequest(t *testing.T) {
	tests := []struct {
		name      string
		timeout   time.Duration
		endpoint  string
		state     any
		questions map[string]Question
		want      string
	}{
		{"zero timeout", 0, "http://unused", "message", testQuestions(), "timeout"},
		{"negative timeout", -time.Second, "http://unused", "message", testQuestions(), "timeout"},
		{"no questions", time.Second, "http://unused", "message", nil, "questions"},
		{"unsupported question", time.Second, "http://unused", "message", map[string]Question{"q": {Type: "choice"}}, "question type"},
		{"unencodable state", time.Second, "http://unused", make(chan int), testQuestions(), "encode"},
		{"invalid URL", time.Second, "://", "message", testQuestions(), "request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := New(Params{Key: "fake-jev-key", Model: "jev", Timeout: tt.timeout})
			client.endpoint = tt.endpoint
			_, err := client.Ask(context.Background(), tt.state, tt.questions)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestClient_Ask_Canceled(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	t.Cleanup(server.Close)
	client := New(Params{Key: "fake-jev-key", Model: "jev", Timeout: time.Second})
	client.endpoint = server.URL
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Ask(ctx, "message", testQuestions())
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, calls.Load())
}
