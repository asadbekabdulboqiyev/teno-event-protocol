package tep

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var testSecret = []byte("0123456789abcdef0123456789abcdef")

func TestEnvelopeValidate(t *testing.T) {
	e := NewEnvelope("order.created", "nescom", map[string]any{"id": "1"})
	if err := e.Validate(); err != nil {
		t.Fatalf("valid envelope rejected: %v", err)
	}
	e2 := NewEnvelope("a.b", "s", map[string]any{})
	e2.Version = "9.0"
	if err := e2.Validate(); err == nil {
		t.Fatal("bad version accepted")
	}
}

func TestSignVerify(t *testing.T) {
	e := NewEnvelope("order.created", "nescom", map[string]any{"x": 1})
	raw, _ := json.Marshal(e)
	sig, err := Sign(e, raw, testSecret)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sig, SigPrefix) {
		t.Fatalf("bad signature prefix: %s", sig)
	}
	if err := Verify(e, raw, sig, testSecret); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	tampered := bytes.Replace(raw, []byte(`"x":1`), []byte(`"x":2`), 1)
	if err := Verify(e, tampered, sig, testSecret); err == nil {
		t.Fatal("tampered body accepted")
	}
}

func TestSignRejectsShortSecret(t *testing.T) {
	e := NewEnvelope("a.b", "s", map[string]any{})
	raw, _ := json.Marshal(e)
	if _, err := Sign(e, raw, []byte("short")); err == nil {
		t.Fatal("short secret accepted")
	}
}

func TestConsumerPushAndIdempotency(t *testing.T) {
	handled := 0
	consumer := NewConsumer(testSecret, func(e *Envelope) (TepCode, error) {
		handled++
		return CodeOK, nil
	})
	server := httptest.NewServer(consumer.PushHandler())
	defer server.Close()

	producer, err := NewClient(ClientOptions{
		URL:        server.URL,
		Secret:     testSecret,
		Key:        "nescom-key",
		Source:     "nescom",
		MaxRetries: 2,
		BaseDelay:  10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	e := NewEnvelope("audit.logged", "placeholder", map[string]any{"action": "login"})
	e.EventID = "00000000-0000-4000-8000-000000000001"
	res1, err := producer.Push(context.Background(), e)
	if err != nil {
		t.Fatalf("first push failed: %v", err)
	}
	if res1.Code != CodeOK {
		t.Fatalf("expected OK, got %s", res1.Code)
	}
	res2, err := producer.Push(context.Background(), e)
	if err != nil {
		t.Fatalf("second push failed: %v", err)
	}
	if res2.Code != CodeDuplicateEvent {
		t.Fatalf("expected DUPLICATE_EVENT, got %s", res2.Code)
	}
	if handled != 1 {
		t.Fatalf("handler called %d times, want 1", handled)
	}
}

func TestConsumerRejectsBadSignature(t *testing.T) {
	consumer := NewConsumer(testSecret, func(e *Envelope) (TepCode, error) {
		return CodeOK, nil
	})
	server := httptest.NewServer(consumer.PushHandler())
	defer server.Close()

	e := NewEnvelope("a.b", "s", map[string]any{})
	raw, _ := json.Marshal(e)
	req, _ := http.NewRequest(http.MethodPost, server.URL, bytes.NewReader(raw))
	req.Header.Set(HeaderSignature, "v1.forgedsignature")
	req.Header.Set(HeaderVersion, e.Version)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestStatusEndpoint(t *testing.T) {
	store := NewMemoryStore(24 * time.Hour)
	consumer := NewConsumer(testSecret, func(e *Envelope) (TepCode, error) { return CodeOK, nil })
	consumer.WithStore(store)

	id := "00000000-0000-4000-8000-000000000777"
	store.Set(id, Record{Status: StatusReceived})

	mux := http.NewServeMux()
	mux.Handle("/v1/events/", http.HandlerFunc(consumer.StatusHandler()))
	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Get(server.URL + "/v1/events/" + id + "/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestMemoryStoreTTL(t *testing.T) {
	store := NewMemoryStore(-time.Second)
	store.Set("k", Record{Status: StatusReceived})
	if _, ok := store.Get("k"); ok {
		t.Fatal("expired entry returned")
	}
}