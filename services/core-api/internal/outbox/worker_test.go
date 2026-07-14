package outbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	events     []Event
	published  []string
	failed     []string
	retryAt    time.Time
	deadLetter string
	lastError  string
}

func (r *fakeRepository) LeaseOutbox(context.Context, string, int, time.Duration) ([]Event, error) {
	return r.events, nil
}
func (r *fakeRepository) MarkOutboxPublished(_ context.Context, _, id string) error {
	r.published = append(r.published, id)
	return nil
}
func (r *fakeRepository) MarkOutboxFailed(_ context.Context, _, id string, retryAt time.Time, lastError, deadLetter string) error {
	r.failed = append(r.failed, id)
	r.retryAt, r.lastError, r.deadLetter = retryAt, lastError, deadLetter
	return nil
}

type fakePublisher struct{ err error }

func (p fakePublisher) Publish(context.Context, Event) error { return p.err }

func TestWorkerMarksPublishedEvents(t *testing.T) {
	repository := &fakeRepository{events: []Event{{ID: "event-1"}}}
	worker := Worker{Repository: repository, Publisher: fakePublisher{}, WorkerID: "worker"}
	count, err := worker.RunOnce(context.Background())
	if err != nil || count != 1 || len(repository.published) != 1 {
		t.Fatalf("unexpected publish result: count=%d published=%v err=%v", count, repository.published, err)
	}
}

func TestWorkerRetriesThenDeadLetters(t *testing.T) {
	publishErr := errors.New("stream unavailable")
	repository := &fakeRepository{events: []Event{{ID: "event-1", RetryCount: 0}}}
	worker := Worker{Repository: repository, Publisher: fakePublisher{err: publishErr}, WorkerID: "worker", MaxAttempts: 2, BaseBackoff: time.Second}
	if _, err := worker.RunOnce(context.Background()); !errors.Is(err, publishErr) {
		t.Fatalf("expected publisher error, got %v", err)
	}
	if repository.retryAt.IsZero() || repository.deadLetter != "" {
		t.Fatalf("first failure should retry: retry=%v dead=%q", repository.retryAt, repository.deadLetter)
	}
	repository.events[0].RetryCount = 1
	if _, err := worker.RunOnce(context.Background()); !errors.Is(err, publishErr) {
		t.Fatalf("expected publisher error, got %v", err)
	}
	if !repository.retryAt.IsZero() || repository.deadLetter != publishErr.Error() {
		t.Fatalf("second failure should dead letter: retry=%v dead=%q", repository.retryAt, repository.deadLetter)
	}
}
