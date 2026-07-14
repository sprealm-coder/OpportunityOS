package audit

import (
	"sync"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

type Record struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenant_id"`
	ActorID    string         `json:"actor_id"`
	Action     string         `json:"action"`
	ObjectType string         `json:"object_type"`
	ObjectID   string         `json:"object_id"`
	RequestID  string         `json:"request_id"`
	TraceID    string         `json:"trace_id"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Log struct {
	mu      sync.RWMutex
	records []Record
}

func (l *Log) Append(record Record) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if record.ID == "" {
		record.ID = platform.NewID("aud")
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	l.records = append(l.records, record)
}

func (l *Log) ForTenant(tenantID string) []Record {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]Record, 0)
	for _, record := range l.records {
		if record.TenantID == tenantID {
			result = append(result, record)
		}
	}
	return result
}
