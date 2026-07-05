// Package httpclient provides small helpers for ZhiCore service-to-service HTTP calls.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const successCode = 200

type Config struct {
	BaseURL       string
	CallerService string
	HTTPClient    *http.Client
	Timeout       time.Duration
}

type Client struct {
	baseURL       string
	callerService string
	httpClient    *http.Client
	timeout       time.Duration
}

type ProviderError struct {
	StatusCode int
	Code       int
}

func (e ProviderError) Error() string {
	return fmt.Sprintf("provider status=%d code=%d", e.StatusCode, e.Code)
}

func New(config Config) *Client {
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:       strings.TrimSpace(config.BaseURL),
		callerService: strings.TrimSpace(config.CallerService),
		httpClient:    httpClient,
		timeout:       config.Timeout,
	}
}

func (c *Client) DoJSON(ctx context.Context, method, path, operation string, body any, out any) error {
	endpoint, err := endpointURL(c.baseURL, path)
	if err != nil {
		return errors.New("build provider request: invalid base url")
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal provider request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return fmt.Errorf("build provider request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Caller-Service", c.callerService)
	req.Header.Set("X-Caller-Operation", operation)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("call provider: %w", err)
		}
		return errors.New("call provider failed")
	}
	defer resp.Body.Close()

	var envelope responseEnvelope
	decodeErr := json.NewDecoder(resp.Body).Decode(&envelope)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if decodeErr != nil {
			return ProviderError{StatusCode: resp.StatusCode}
		}
		return ProviderError{StatusCode: resp.StatusCode, Code: envelope.Code}
	}
	if decodeErr != nil {
		return errors.New("decode provider envelope failed")
	}
	if envelope.Code != successCode {
		return ProviderError{StatusCode: resp.StatusCode, Code: envelope.Code}
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return errors.New("decode provider data failed")
	}
	return nil
}

type responseEnvelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

func endpointURL(baseURL, path string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid base url")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
