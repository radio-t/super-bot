// Package jev provides bounded requests for TypeSafe's Noul decision API.
package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxResponseBytes = 64 * 1024

// Params configures a Jev client.
type Params struct {
	// Key is the bearer credential.
	Key string
	// Model identifies the Jev model to query.
	Model string
	// Timeout bounds the request, including reading the response body.
	Timeout time.Duration
}

// Question describes one Noul judgment.
type Question struct {
	// Type must be "noul".
	Type string `json:"type"`
	// Instructions describes the judgment to make about the state.
	Instructions string `json:"instructions"`
	// Criteria optionally defines the meanings of "true" and "false".
	Criteria map[string]string `json:"criteria,omitempty"`
}

// Answer contains one Noul probability.
type Answer struct {
	// Type identifies the returned primitive.
	Type string `json:"type"`
	// Noul is the probability of true; nil distinguishes missing/null from zero.
	Noul *float64 `json:"noul"`
}

// Response contains validated answers and the resolved model name.
type Response struct {
	// Model is the model that answered, which may resolve a requested alias.
	Model string `json:"model"`
	// Answers maps each requested question ID to its answer.
	Answers map[string]Answer `json:"answers"`
}

// Client sends requests without retries or redirects.
type Client struct {
	key      string
	model    string
	endpoint string
	http     *http.Client
}

// New creates a client; Ask rejects a non-positive timeout.
func New(params Params) *Client {
	return &Client{
		key:      params.Key,
		model:    params.Model,
		endpoint: "https://api.typesafe.ai/v1/systemone",
		http: &http.Client{
			Timeout: params.Timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Ask evaluates all questions in one bounded request and validates every requested answer.
func (c *Client) Ask(ctx context.Context, state any, questions map[string]Question) (Response, error) {
	if c.http.Timeout <= 0 {
		return Response{}, fmt.Errorf("jev timeout must be positive")
	}
	if len(questions) == 0 {
		return Response{}, fmt.Errorf("jev questions must not be empty")
	}
	for id, question := range questions {
		if question.Type != "noul" {
			return Response{}, fmt.Errorf("unsupported jev question type for %q", id)
		}
	}

	payload := struct {
		Model     string              `json:"model"`
		State     any                 `json:"state"`
		Questions map[string]Question `json:"questions"`
	}{Model: c.model, State: state, Questions: questions}
	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, fmt.Errorf("encode jev request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return Response{}, fmt.Errorf("create jev request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("send jev request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return Response{}, fmt.Errorf("read jev response: %w", err)
	}
	if len(data) > maxResponseBytes {
		return Response{}, fmt.Errorf("jev response too large (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return Response{}, c.httpError(resp.StatusCode, data)
	}

	var result Response
	if err := json.Unmarshal(data, &result); err != nil {
		return Response{}, fmt.Errorf("decode jev response: %w", err)
	}
	if err := c.checkAnswers(result, questions); err != nil {
		return Response{}, err
	}
	return result, nil
}

func (c *Client) checkAnswers(resp Response, questions map[string]Question) error {
	if resp.Model == "" {
		return fmt.Errorf("jev response missing model")
	}
	for id, question := range questions {
		answer, ok := resp.Answers[id]
		if !ok {
			return fmt.Errorf("jev response missing answer for %q", id)
		}
		if answer.Type != question.Type {
			return fmt.Errorf("invalid jev answer type for %q", id)
		}
		if answer.Noul == nil || *answer.Noul < 0 || *answer.Noul > 1 {
			return fmt.Errorf("invalid jev noul for %q", id)
		}
	}
	return nil
}

func (c *Client) httpError(status int, data []byte) error {
	var detail struct {
		Detail json.RawMessage `json:"detail"`
	}
	if err := json.Unmarshal(data, &detail); err != nil {
		return fmt.Errorf("jev HTTP %d", status)
	}
	errorType, message := c.errorDetail(detail.Detail)
	text := strings.Trim(errorType+": "+message, ": ")
	if c.key != "" {
		text = strings.ReplaceAll(text, c.key, "[redacted]")
	}
	text = strings.Join(strings.Fields(text), " ")
	runes := []rune(text)
	if len(runes) > 256 {
		text = string(runes[:256]) + "..."
	}
	if text == "" {
		return fmt.Errorf("jev HTTP %d", status)
	}
	return fmt.Errorf("jev HTTP %d: %s", status, text)
}

func (c *Client) errorDetail(data json.RawMessage) (errorType, message string) {
	var object struct {
		Type    string `json:"error_type"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &object); err == nil && (object.Type != "" || object.Message != "") {
		return object.Type, object.Message
	}
	var list []struct {
		Type    string `json:"type"`
		Message string `json:"msg"`
	}
	if err := json.Unmarshal(data, &list); err == nil && len(list) > 0 {
		return list[0].Type, list[0].Message
	}
	if err := json.Unmarshal(data, &message); err != nil {
		return "", ""
	}
	return "", message
}
