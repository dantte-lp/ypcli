// Package api is the context-aware HTTP transport for the yopass server. It
// adds Bearer authentication, typed error classification and JSON envelope
// handling on top of net/http. Timeouts are driven by the caller's context.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client talks to a single yopass API base URL.
type Client struct {
	base  string
	token string
	http  *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithToken sets the Bearer token sent on every request.
func WithToken(token string) Option {
	return func(c *Client) { c.token = token }
}

// WithHTTPClient overrides the underlying *http.Client (used in tests).
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// New builds a Client for the given API base URL.
func New(baseURL string, opts ...Option) *Client {
	c := &Client{
		base: strings.TrimSuffix(baseURL, "/"),
		http: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// envelope is the standard {"message": ...} response body.
type envelope struct {
	Message string `json:"message"`
}

// do issues a request, applying auth and headers, and returns the raw
// response. Transport errors are wrapped as ErrNetwork.
func (c *Client) do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNetwork, err)
	}
	return resp, nil
}

// message issues a request expecting a JSON {"message": ...} response and
// returns the message value. Non-200 responses become a typed *Error.
func (c *Client) message(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (string, error) {
	resp, err := c.do(ctx, method, path, body, headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", serverError(resp.StatusCode, raw)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	return env.Message, nil
}

// serverError builds a typed *Error, preferring the server's {"message": ...}
// text when present.
func serverError(status int, body []byte) error {
	msg := strings.TrimSpace(string(body))
	var env envelope
	if json.Unmarshal(body, &env) == nil && env.Message != "" {
		msg = env.Message
	}
	return &Error{Status: status, Message: msg, kind: classify(status)}
}
