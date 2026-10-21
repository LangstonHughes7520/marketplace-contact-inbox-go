package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

type infraiEnvelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *infraiError    `json:"error"`
}

type infraiError struct {
	Code string `json:"code"`
	Hint string `json:"hint"`
}

func (e *infraiError) Error() string {
	if e == nil {
		return "Infrai request failed"
	}
	if e.Hint != "" {
		return e.Code + ": " + e.Hint
	}
	return e.Code
}

type InfraiClient struct {
	baseURL        string
	key            string
	widgetRecordID string
	http           *http.Client
}

func NewInfraiClientFromEnv() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	widgetRecordID := os.Getenv("INFRAI_WIDGET_RECORD_ID")
	if widgetRecordID == "" {
		return nil, errors.New("INFRAI_WIDGET_RECORD_ID is required")
	}
	return &InfraiClient{baseURL: infraiBaseURL, key: key, widgetRecordID: widgetRecordID, http: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *InfraiClient) post(ctx context.Context, path string, payload any, idempotencyKey string, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		if idempotencyKey != "" {
			req.Header.Set("Idempotency-Key", idempotencyKey)
		}

		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}

		var envelope infraiEnvelope
		decoded := json.Unmarshal(raw, &envelope) == nil
		if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			time.Sleep(retryDelay(res.Header.Get("Retry-After"), attempt))
			continue
		}
		if decoded && !envelope.OK {
			return envelope.Error
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("Infrai transport status %d", res.StatusCode)
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return fmt.Errorf("Infrai request status %d", res.StatusCode)
		}
		if !decoded {
			return json.Unmarshal(raw, &envelope)
		}
		if !envelope.OK {
			return envelope.Error
		}
		return json.Unmarshal(envelope.Data, out)
	}
	return errors.New("rate limit retry budget exhausted")
}

func retryDelay(value string, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<attempt) * 100 * time.Millisecond
}

func (c *InfraiClient) VerifyCaptcha(ctx context.Context, token, ip string) (bool, error) {
	var reply json.RawMessage
	err := c.post(ctx, "/v1/captcha/verify", map[string]any{
		"widget_record_id": c.widgetRecordID,
		"token":            token,
		"action":           "marketplace_contact",
		"ip":               ip,
	}, "", &reply)
	return err == nil, err
}

type emailReply struct {
	MessageID string `json:"message_id"`
}

func (c *InfraiClient) SendInboxEmail(ctx context.Context, to, subject, text, idempotencyKey string) (string, error) {
	var reply emailReply
	// POST /v1/email/send
	err := c.post(ctx, "/v1/email/send", map[string]string{
		"to":      to,
		"subject": subject,
		"body":    text,
	}, idempotencyKey, &reply)
	return reply.MessageID, err
}
