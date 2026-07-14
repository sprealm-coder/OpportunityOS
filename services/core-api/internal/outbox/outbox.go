package outbox

import (
	"context"
	"errors"
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
	RetryCount    int            `json:"retry_count"`
}

type Repository interface {
	LeaseOutbox(context.Context, string, int, time.Duration) ([]Event, error)
	MarkOutboxPublished(context.Context, string, string) error
	MarkOutboxFailed(context.Context, string, string, time.Time, string, string) error
}

type Publisher interface {
	Publish(context.Context, Event) error
}

type Worker struct {
	Repository  Repository
	Publisher   Publisher
	WorkerID    string
	BatchSize   int
	Lease       time.Duration
	MaxAttempts int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

func (w Worker) RunOnce(ctx context.Context) (int, error) {
	batchSize, lease, maxAttempts := w.BatchSize, w.Lease, w.MaxAttempts
	if batchSize <= 0 {
		batchSize = 50
	}
	if lease <= 0 {
		lease = 30 * time.Second
	}
	if maxAttempts <= 0 {
		maxAttempts = 8
	}
	events, err := w.Repository.LeaseOutbox(ctx, w.WorkerID, batchSize, lease)
	if err != nil {
		return 0, err
	}
	var failures []error
	for _, event := range events {
		if err = w.Publisher.Publish(ctx, event); err == nil {
			if markErr := w.Repository.MarkOutboxPublished(ctx, w.WorkerID, event.ID); markErr != nil {
				failures = append(failures, markErr)
			}
			continue
		}
		attempt := event.RetryCount + 1
		retryAt, deadLetterReason := time.Now().UTC().Add(w.backoff(attempt)), ""
		if attempt >= maxAttempts {
			retryAt = time.Time{}
			deadLetterReason = err.Error()
		}
		if markErr := w.Repository.MarkOutboxFailed(ctx, w.WorkerID, event.ID, retryAt, err.Error(), deadLetterReason); markErr != nil {
			failures = append(failures, errors.Join(err, markErr))
		} else {
			failures = append(failures, err)
		}
	}
	return len(events), errors.Join(failures...)
}

func (w Worker) backoff(attempt int) time.Duration {
	base, maximum := w.BaseBackoff, w.MaxBackoff
	if base <= 0 {
		base = time.Second
	}
	if maximum <= 0 {
		maximum = 5 * time.Minute
	}
	delay := base
	for index := 1; index < attempt && delay < maximum; index++ {
		delay *= 2
	}
	if delay > maximum {
		return maximum
	}
	return delay
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
