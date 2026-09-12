package tep

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HandlerFunc func(e *Envelope) (TepCode, error)

type ConsumerOptions struct {
	Secret []byte
	Store  Store
}

type Consumer struct {
	secret []byte
	store  Store
	handle HandlerFunc
}

func NewConsumer(secret []byte, handle HandlerFunc) *Consumer {
	return &Consumer{secret: secret, store: NewMemoryStore(), handle: handle}
}

func (c *Consumer) WithStore(store Store) *Consumer {
	c.store = store
	return c
}

func (c *Consumer) PushHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, Result{Code: CodeBadRequest, Error: "method not allowed"})
			return
		}
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, Result{Code: CodeBadRequest, Error: err.Error()})
			return
		}
		var e Envelope
		if err := json.Unmarshal(rawBody, &e); err != nil {
			writeJSON(w, http.StatusBadRequest, Result{Code: CodeBadRequest, Error: err.Error()})
			return
		}
		if err := e.Validate(); err != nil {
			writeJSON(w, http.StatusBadRequest, Result{Code: CodeBadRequest, Error: err.Error()})
			return
		}
		if v := r.Header.Get(HeaderVersion); v != "" && v != e.Version {
			writeJSON(w, http.StatusUnsupportedMediaType, Result{Code: CodeVersionUnsupported, Error: fmt.Sprintf("expected version %s", e.Version)})
			return
		}
		sig := r.Header.Get(HeaderSignature)
		if sig == "" {
			writeJSON(w, http.StatusUnauthorized, Result{Code: CodeSigInvalid, Error: "missing signature header"})
			return
		}
		if err := Verify(&e, rawBody, sig, c.secret); err != nil {
			writeJSON(w, http.StatusUnauthorized, Result{Code: CodeSigInvalid, Error: err.Error()})
			return
		}
		key := e.DedupeKey()
		if rec, ok := c.store.Get(key); ok {
			if rec.Status == StatusFailed {
				writeJSON(w, http.StatusBadGateway, Result{Code: CodeUpstreamError, EventID: e.EventID, Status: rec.Status, Error: rec.Error})
			} else {
				writeJSON(w, http.StatusOK, Result{Code: CodeDuplicateEvent, EventID: e.EventID, Status: rec.Status})
			}
			return
		}
		c.store.Set(key, Record{Status: StatusReceived})
		code, err := c.handle(&e)
		if err != nil {
			c.store.Set(key, Record{Status: StatusFailed, Error: err.Error()})
			writeJSON(w, http.StatusBadGateway, Result{Code: CodeUpstreamError, EventID: e.EventID, Status: StatusFailed, Error: err.Error()})
			return
		}
		status := StatusDelivered
		if code == CodeEventAccepted {
			status = StatusProcessing
		}
		c.store.Set(key, Record{Status: status, ProcessedAt: time.Now().UTC()})
		writeJSON(w, httpStatus(code), Result{Code: code, EventID: e.EventID, Status: status})
	}
}

func (c *Consumer) StatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/v1/events/")
		id = strings.TrimSuffix(id, "/status")
		id = strings.TrimPrefix(id, "/")
		rec, ok := c.store.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, Result{Code: CodeEventNotFound, EventID: id})
			return
		}
		code := CodeOK
		if rec.Status == StatusFailed {
			code = CodeUpstreamError
		}
		writeJSON(w, http.StatusOK, Result{Code: code, EventID: id, Status: rec.Status, ProcessedAt: rec.ProcessedAt, Error: rec.Error})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}