package inbox

import (
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"sync"
	"time"
)

type Event struct {
	ID, TenantID, ExternalEventID, IdempotencyKey string
	Payload                                       map[string]any
	ReceivedAt                                    time.Time
}
type Memory struct {
	mu          sync.Mutex
	external    map[string]bool
	idempotency map[string]bool
	events      []Event
}

func New() *Memory { return &Memory{external: map[string]bool{}, idempotency: map[string]bool{}} }
func (m *Memory) Accept(event Event) (bool, error) {
	if event.TenantID == "" {
		return false, platform.ErrTenantRequired
	}
	if event.ExternalEventID == "" || event.IdempotencyKey == "" {
		return false, platform.Invalid("invalid_event", "external event ID and idempotency key are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	externalKey := event.TenantID + "/" + event.ExternalEventID
	idemKey := event.TenantID + "/" + event.IdempotencyKey
	if m.external[externalKey] || m.idempotency[idemKey] {
		return false, nil
	}
	event.ID = platform.NewID("inbox")
	event.ReceivedAt = time.Now().UTC()
	m.external[externalKey] = true
	m.idempotency[idemKey] = true
	m.events = append(m.events, event)
	return true, nil
}
