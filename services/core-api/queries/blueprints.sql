-- name: CreateBusinessBlueprint :one
INSERT INTO business_blueprints (
  tenant_id, source_opportunity_id, name, description,
  version, status, definition, created_by
) VALUES ($1, $2, $3, $4, 1, 'draft', $5, $6)
RETURNING *;

-- name: GetBusinessBlueprint :one
SELECT * FROM business_blueprints WHERE tenant_id = $1 AND id = $2;

-- name: LockBusinessBlueprint :one
SELECT * FROM business_blueprints WHERE tenant_id = $1 AND id = $2 FOR UPDATE;

-- name: ListBusinessBlueprints :many
SELECT * FROM business_blueprints
WHERE tenant_id = $1
ORDER BY updated_at DESC, id
LIMIT $2;

-- name: UpdateBusinessBlueprint :one
UPDATE business_blueprints
SET status = $3, version = $4, definition = $5,
    approved_by = $6, updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;
