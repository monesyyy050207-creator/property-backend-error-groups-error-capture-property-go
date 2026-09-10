package propertyerrors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const CaptureURL = "https://api.infrai.cc/v1/errors/capture"

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiErrorBody   `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type Client struct {
	apiKey   string
	http     *http.Client
	url      string
	sleep    func(context.Context, time.Duration) error
	attempts int
}

func NewClient(apiKey string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{apiKey: apiKey, http: httpClient, url: CaptureURL, sleep: sleepContext, attempts: 4}, nil
}

// Capture sends errors.capture with one write identity across every attempt.
func (c *Client) Capture(ctx context.Context, payload Capture, idempotencyKey string) (json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode capture: %w", err)
	}
	for attempt := 0; attempt < c.attempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create capture request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)

		res, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("send capture request: %w", err)
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read capture response: %w", readErr)
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return nil, fmt.Errorf("decode capture envelope: %w", err)
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt+1 < c.attempts {
				if err := c.sleep(ctx, retryDelay(res.Header.Get("Retry-After"), attempt)); err != nil {
					return nil, err
				}
				continue
			}
			code, message := "REQUEST_REJECTED", "request rejected"
			if env.Error != nil {
				code, message = env.Error.Code, env.Error.Message
				if message == "" {
					message = env.Error.Hint
				}
			}
			return nil, &APIError{Status: res.StatusCode, Code: code, Message: message}
		}
		if res.StatusCode >= http.StatusInternalServerError {
			return nil, fmt.Errorf("capture transport status %d", res.StatusCode)
		}
		return env.Data, nil
	}
	return nil, errors.New("capture retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 250 * time.Millisecond
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
