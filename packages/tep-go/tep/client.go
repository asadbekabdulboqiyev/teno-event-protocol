package tep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	HeaderVersion   = "X-TEP-Version"
	HeaderSignature = "X-TEP-Signature"
	HeaderKey       = "X-TEP-Key"
)

type Result struct {
	Code        TepCode   `json:"code"`
	EventID     string    `json:"event_id,omitempty"`
	Status      Status    `json:"status,omitempty"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type Client struct {
	url        string
	secret     []byte
	key        string
	source     string
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	http       *http.Client
}

type ClientOptions struct {
	URL        string
	Secret     []byte
	Key        string
	Source     string
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	HTTP       *http.Client
}

func NewClient(opts ClientOptions) (*Client, error) {
	if len(opts.Secret) < 32 {
		return nil, fmt.Errorf("secret must be at least 32 bytes")
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 5
	}
	if opts.BaseDelay == 0 {
		opts.BaseDelay = time.Second
	}
	if opts.MaxDelay == 0 {
		opts.MaxDelay = 60 * time.Second
	}
	if opts.HTTP == nil {
		opts.HTTP = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{
		url:        opts.URL,
		secret:     opts.Secret,
		key:        opts.Key,
		source:     opts.Source,
		maxRetries: opts.MaxRetries,
		baseDelay:  opts.BaseDelay,
		maxDelay:   opts.MaxDelay,
		http:       opts.HTTP,
	}, nil
}

func (c *Client) Push(ctx context.Context, e *Envelope) (Result, error) {
	e.Source = c.source
	rawBody, err := json.Marshal(e)
	if err != nil {
		return Result{}, err
	}
	sig, err := Sign(e, rawBody, c.secret)
	if err != nil {
		return Result{}, err
	}
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.baseDelay << uint(attempt-1)
			if delay > c.maxDelay {
				delay = c.maxDelay
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return Result{}, ctx.Err()
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(rawBody))
		if err != nil {
			return Result{}, err
		}
		req.Header.Set("content-type", "application/json")
		req.Header.Set(HeaderVersion, e.Version)
		req.Header.Set(HeaderKey, c.key)
		req.Header.Set(HeaderSignature, sig)
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		var result Result
		if err := json.Unmarshal(body, &result); err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return result, nil
			}
			lastErr = fmt.Errorf("TEP push failed (%d): %s", resp.StatusCode, string(body))
			continue
		}
		lastErr = fmt.Errorf("TEP push bad response (%d)", resp.StatusCode)
	}
	return Result{}, fmt.Errorf("TEP push failed after %d attempts: %w", c.maxRetries+1, lastErr)
}