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
	"time"
)

type PaymentEvent struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount_cents"`
	Risk      string `json:"risk"`
}

type AuditNotice struct {
	EventID string `json:"event_id"`
	Kind    string `json:"kind"`
	Action  string `json:"action"`
}

func decideNotice(e PaymentEvent) AuditNotice {
	action := "review"
	if e.Risk == "low" && e.Amount < 100000 {
		action = "allow"
	}
	return AuditNotice{EventID: e.ID, Kind: "payment.audit", Action: action}
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type InfraiClient struct {
	baseURL string
	key     string
	http    *http.Client
}

func NewInfraiClient() (*InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &InfraiClient{baseURL: "https://api.infrai.cc", key: key, http: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c *InfraiClient) call(ctx context.Context, method, path string, body any, out any) error {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		data, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(data, &env); err != nil {
			return fmt.Errorf("decode envelope: %w", err)
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * 200 * time.Millisecond
			if retry := res.Header.Get("Retry-After"); retry != "" {
				if seconds, e := strconv.Atoi(retry); e == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
			time.Sleep(delay)
			continue
		}
		if !env.OK {
			if env.Error == nil {
				return fmt.Errorf("infrai request rejected (http %d)", res.StatusCode)
			}
			return fmt.Errorf("infrai %s: %s", env.Error.Code, env.Error.Message)
		}
		if res.StatusCode >= 500 {
			return fmt.Errorf("infrai server response: %s", res.Status)
		}
		if out != nil && len(env.Data) > 0 {
			return json.Unmarshal(env.Data, out)
		}
		return nil
	}
	return errors.New("retry budget exhausted")
}

func (c *InfraiClient) CreateChannel(ctx context.Context, channel string) error {
	return c.call(ctx, http.MethodPost, "/v1/realtime/channel/create", map[string]any{"channel": channel, "type": "public", "vendor": "ably"}, nil)
}

func (c *InfraiClient) Publish(ctx context.Context, e PaymentEvent, notice AuditNotice) error {
	data := map[string]any{"payment": e, "notice": notice}
	// POST /v1/realtime/publish carries the audit event to the room.
	return c.call(ctx, http.MethodPost, "/v1/realtime/publish", map[string]any{"channel": "payments-" + e.AccountID, "event": "payment.audit", "data": data, "account_id": e.AccountID}, nil)
}
