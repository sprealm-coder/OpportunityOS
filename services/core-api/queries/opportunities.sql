-- name: CreateOpportunity :one
INSERT INTO opportunities (tenant_id,name,description,status,created_by)
VALUES ($1,$2,$3,'detected',$4)
RETURNING *;

-- name: GetOpportunity :one
SELECT * FROM opportunities WHERE tenant_id=$1 AND id=$2;

-- name: ListOpportunities :many
SELECT * FROM opportunities WHERE tenant_id=$1 ORDER BY created_at DESC LIMIT $2;

-- name: AppendOutboxEvent :exec
INSERT INTO outbox_events (tenant_id,aggregate_type,aggregate_id,event_type,aggregate_version,trace_id,payload)
VALUES ($1,$2,$3,$4,$5,$6,$7);

