package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/inbox"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/outbox"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/platform"
	"github.com/opportunity-os/opportunity-os/services/core-api/internal/tenancy"
)

func (s *Store) LeaseOutbox(ctx context.Context, workerID string, limit int, lease time.Duration) ([]outbox.Event, error) {
	if workerID == "" || limit <= 0 || lease <= 0 {
		return nil, platform.Invalid("invalid_outbox_lease", "worker ID, positive limit, and lease duration are required")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		WITH candidates AS (
			SELECT id FROM outbox_events
			WHERE processed_at IS NULL AND dead_letter_reason IS NULL
			  AND (next_retry_at IS NULL OR next_retry_at<=now())
			  AND (locked_until IS NULL OR locked_until<now())
			ORDER BY occurred_at,id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		)
		UPDATE outbox_events event
		SET locked_by=$1,locked_until=$3
		FROM candidates WHERE event.id=candidates.id
		RETURNING event.id,event.tenant_id,event.aggregate_type,event.aggregate_id,event.event_type,
		          event.aggregate_version,event.trace_id,event.payload,event.occurred_at,event.processed_at,event.retry_count`,
		workerID, limit, time.Now().UTC().Add(lease))
	if err != nil {
		return nil, mapError(err)
	}
	defer rows.Close()
	events := []outbox.Event{}
	for rows.Next() {
		var event outbox.Event
		var payload []byte
		if err = rows.Scan(&event.ID, &event.TenantID, &event.AggregateType, &event.AggregateID, &event.EventType,
			&event.Version, &event.TraceID, &payload, &event.OccurredAt, &event.ProcessedAt, &event.RetryCount); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(payload, &event.Payload); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, mapError(err)
	}
	return events, nil
}

func (s *Store) MarkOutboxPublished(ctx context.Context, workerID, eventID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE outbox_events SET processed_at=now(),published_at=now(),locked_by=NULL,locked_until=NULL,last_error=NULL
		WHERE id=$1 AND locked_by=$2 AND processed_at IS NULL`, eventID, workerID)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return platform.Invalid("outbox_lease_lost", "outbox event lease is no longer owned by this worker")
	}
	return nil
}

func (s *Store) MarkOutboxFailed(ctx context.Context, workerID, eventID string, retryAt time.Time, lastError, deadLetterReason string) error {
	var retry any
	if !retryAt.IsZero() {
		retry = retryAt
	}
	var deadLetter any
	if deadLetterReason != "" {
		deadLetter = deadLetterReason
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE outbox_events
		SET retry_count=retry_count+1,next_retry_at=$3,dead_letter_reason=$4,last_error=$5,locked_by=NULL,locked_until=NULL
		WHERE id=$1 AND locked_by=$2 AND processed_at IS NULL`, eventID, workerID, retry, deadLetter, lastError)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() != 1 {
		return platform.Invalid("outbox_lease_lost", "outbox event lease is no longer owned by this worker")
	}
	return nil
}

func (s *Store) AcceptInbox(ctx context.Context, scope tenancy.Scope, event inbox.Event) (bool, error) {
	if event.ExternalEventID == "" || event.IdempotencyKey == "" {
		return false, platform.Invalid("invalid_event", "external event ID and idempotency key are required")
	}
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return false, err
	}
	return runTenantRead(ctx, s, scope.TenantID, func(tx pgx.Tx) (bool, error) {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO inbox_events (tenant_id,external_event_id,idempotency_key,payload)
			VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING RETURNING id`,
			scope.TenantID, event.ExternalEventID, event.IdempotencyKey, payload).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return err == nil, err
	})
}

var _ outbox.Repository = (*Store)(nil)
