-- name: CreateAuditRecord :one
INSERT INTO audit_log (
  tenant_id, actor_id, action, object_type, object_id,
  request_id, trace_id, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListAuditRecords :many
SELECT * FROM audit_log
WHERE tenant_id = $1
ORDER BY created_at DESC, id
LIMIT $2;

-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (
  tenant_id, aggregate_type, aggregate_id, event_type,
  aggregate_version, trace_id, payload
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ReserveCommand :one
INSERT INTO command_idempotency (
  tenant_id, idempotency_key, operation
) VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING
RETURNING tenant_id, idempotency_key, operation;

-- name: GetCommand :one
SELECT operation, aggregate_id, response
FROM command_idempotency
WHERE tenant_id = $1 AND idempotency_key = $2;

-- name: CompleteCommand :exec
UPDATE command_idempotency
SET aggregate_id = $3, response = $4
WHERE tenant_id = $1 AND idempotency_key = $2;
