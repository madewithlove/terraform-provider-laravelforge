// Package forge provides a thin client for the Laravel Forge JSON:API
// (the organization-scoped API documented at
// https://forge.laravel.com/api/docs).
//
// The client deliberately stays small: callers pass plain request bodies and
// receive the unwrapped JSON:API "data" member, leaving resource-specific
// marshaling to the Terraform resources.
package forge

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

// DefaultEndpoint is the base URL of the public Laravel Forge API.
const DefaultEndpoint = "https://forge.laravel.com/api"

// Client talks to the Laravel Forge API using a bearer token.
type Client struct {
	HTTPClient *http.Client
	Endpoint   string
	Token      string
}

// New returns a Client for the given endpoint and token. An empty endpoint
// falls back to DefaultEndpoint.
func New(endpoint, token string) *Client {
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	return &Client{
		HTTPClient: &http.Client{Timeout: 60 * time.Second},
		Endpoint:   strings.TrimRight(endpoint, "/"),
		Token:      token,
	}
}

// APIError describes a non-2xx response from the Forge API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("forge API returned HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("forge API returned HTTP %d: %s", e.StatusCode, e.Body)
}

// NotFound reports whether err is an APIError with a 404 status.
func NotFound(err error) bool {
	var apiErr *APIError
	if e, ok := err.(*APIError); ok {
		apiErr = e
	}
	return apiErr != nil && apiErr.StatusCode == http.StatusNotFound
}

// Do performs an HTTP request against the API. body, if non-nil, is JSON
// encoded and sent as the request payload. On a 2xx response the JSON:API
// "data" member is decoded into out when out is non-nil; a response without a
// "data" envelope is decoded into out directly. Non-2xx responses return an
// *APIError.
func (c *Client) Do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.Endpoint+path, reader)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(payload))}
	}

	if out == nil || len(payload) == 0 {
		return nil
	}

	// Successful payloads are wrapped in a JSON:API "data" envelope; unwrap it
	// when present, otherwise decode the body directly.
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil && len(envelope.Data) > 0 {
		return json.Unmarshal(envelope.Data, out)
	}
	return json.Unmarshal(payload, out)
}
