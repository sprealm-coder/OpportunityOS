-- name: CreateOpportunity :one
INSERT INTO opportunities (tenant_id, name, description, status, created_by)
VALUES ($1, $2, $3, 'detected', $4)
RETURNING *;

-- name: GetOpportunity :one
SELECT * FROM opportunities WHERE tenant_id = $1 AND id = $2;

-- name: LockOpportunity :one
SELECT * FROM opportunities WHERE tenant_id = $1 AND id = $2 FOR UPDATE;

-- name: ListOpportunities :many
SELECT * FROM opportunities
WHERE tenant_id = $1
ORDER BY updated_at DESC, id
LIMIT $2;

-- name: UpdateOpportunity :one
UPDATE opportunities
SET status = $3, score = $4, version = $5, updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;

-- name: CreateOpportunityEvidence :one
INSERT INTO opportunity_evidence (
  tenant_id, opportunity_id, kind, summary, confidence
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListOpportunityEvidence :many
SELECT * FROM opportunity_evidence
WHERE tenant_id = $1 AND opportunity_id = $2
ORDER BY created_at, id;

-- name: CreateOpportunityReview :one
INSERT INTO opportunity_reviews (
  tenant_id, opportunity_id, actor_id, decision, rationale
) VALUES ($1, $2, $3, $4, $5)
RETURNING *;
