package tep

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

const (
	Protocol = "tep"
	Version  = "1.0"

	SigPrefix = "v1."
)

type Envelope struct {
	Protocol       string         `json:"protocol"`
	Version        string         `json:"version"`
	EventID        string         `json:"event_id"`
	Type           string         `json:"type"`
	Source         string         `json:"source"`
	Timestamp      string         `json:"timestamp"`
	CorrelationID  string         `json:"correlation_id,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Payload        map[string]any `json:"payload"`
}

func NewEnvelope(typeName, source string, payload map[string]any) *Envelope {
	return &Envelope{
		Protocol:  Protocol,
		Version:   Version,
		EventID:   NewUUID(),
		Type:      typeName,
		Source:    source,
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.999Z"),
		Payload:   payload,
	}
}

func (e *Envelope) Validate() error {
	if e.Protocol != Protocol {
		return fmt.Errorf("protocol must be %q", Protocol)
	}
	if e.Version != Version {
		return fmt.Errorf("unsupported version %q", e.Version)
	}
	if e.EventID == "" {
		return fmt.Errorf("event_id required")
	}
	if e.Type == "" {
		return fmt.Errorf("type required")
	}
	if e.Source == "" {
		return fmt.Errorf("source required")
	}
	if _, err := time.Parse(time.RFC3339Nano, e.Timestamp); err != nil {
		return fmt.Errorf("timestamp must be ISO8601: %w", err)
	}
	if e.Payload == nil {
		return fmt.Errorf("payload must be an object")
	}
	return nil
}

func (e *Envelope) DedupeKey() string {
	if e.IdempotencyKey != "" {
		return e.IdempotencyKey
	}
	return e.EventID
}

func Canonical(e *Envelope, rawBody []byte) []byte {
	return []byte(fmt.Sprintf("tep\n%s\n%s\n%s\n%s\n%s\n%s",
		Version, e.EventID, e.Timestamp, e.Source, e.Type, string(rawBody)))
}

func Sign(e *Envelope, rawBody []byte, secret []byte) (string, error) {
	if len(secret) < 32 {
		return "", fmt.Errorf("secret must be at least 32 bytes")
	}
	mac := hmac.New(sha256.New, secret)
	_, err := mac.Write(Canonical(e, rawBody))
	if err != nil {
		return "", err
	}
	return SigPrefix + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func Verify(e *Envelope, rawBody []byte, signature string, secret []byte) error {
	if len(secret) < 32 {
		return fmt.Errorf("secret must be at least 32 bytes")
	}
	if len(signature) < len(SigPrefix) || signature[:len(SigPrefix)] != SigPrefix {
		return fmt.Errorf("malformed signature")
	}
	provided, err := base64.RawURLEncoding.DecodeString(signature[len(SigPrefix):])
	if err != nil {
		return fmt.Errorf("malformed signature: %w", err)
	}
	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write(Canonical(e, rawBody)); err != nil {
		return err
	}
	expected := mac.Sum(nil)
	if !hmac.Equal(provided, expected) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}