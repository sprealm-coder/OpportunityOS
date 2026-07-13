package outbox

import (
	"sync"
	"time"

	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
)

type Event struct {
	ID            string         `json:"id"`
	TenantID      string         `json:"tenant_id"`
	AggregateType string         `json:"aggregate_type"`
	AggregateID   string         `json:"aggregate_id"`
	EventType     string         `json:"event_type"`
	Version       int            `json:"version"`
	TraceID       string         `json:"trace_id"`
	Payload       map[string]any `json:"payload"`
	OccurredAt    time.Time      `json:"occurred_at"`
	ProcessedAt   *time.Time     `json:"processed_at,omitempty"`
}

type Memory struct {
	mu     sync.RWMutex
	events []Event
}

func (m *Memory) Append(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if event.ID == "" {
		event.ID = platform.NewID("evt")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	m.events = append(m.events, event)
}

func (m *Memory) Pending(tenantID string) []Event {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Event, 0)
	for _, event := range m.events {
		if event.TenantID == tenantID && event.ProcessedAt == nil {
			result = append(result, event)
		}
	}
	return result
}
