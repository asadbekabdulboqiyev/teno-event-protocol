package tep

import (
	"time"
)

type Status string

const (
	StatusReceived   Status = "received"
	StatusProcessing Status = "processing"
	StatusDelivered  Status = "delivered"
	StatusFailed     Status = "failed"
)

type Record struct {
	Status      Status    `json:"status"`
	ProcessedAt time.Time `json:"processed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
}

type Store interface {
	Get(key string) (Record, bool)
	Set(key string, record Record)
}

type MemoryStore struct {
	ttl     time.Duration
	records map[string]memoryEntry
}

type memoryEntry struct {
	record  Record
	expires time.Time
}

func NewMemoryStore(ttl ...time.Duration) *MemoryStore {
	t := 24 * time.Hour
	if len(ttl) > 0 {
		t = ttl[0]
	}
	return &MemoryStore{ttl: t, records: make(map[string]memoryEntry)}
}

func (s *MemoryStore) Get(key string) (Record, bool) {
	entry, ok := s.records[key]
	if !ok {
		return Record{}, false
	}
	if time.Now().After(entry.expires) {
		delete(s.records, key)
		return Record{}, false
	}
	return entry.record, true
}

func (s *MemoryStore) Set(key string, record Record) {
	s.records[key] = memoryEntry{record: record, expires: time.Now().Add(s.ttl)}
}