-- name: CreateIncubationProject :one
INSERT INTO incubation_projects (
  tenant_id, opportunity_id, name, status
) VALUES ($1, $2, $3, 'draft')
RETURNING *;

-- name: GetIncubationProject :one
SELECT * FROM incubation_projects WHERE tenant_id = $1 AND id = $2;

-- name: LockIncubationProject :one
SELECT * FROM incubation_projects WHERE tenant_id = $1 AND id = $2 FOR UPDATE;

-- name: ListIncubationProjects :many
SELECT * FROM incubation_projects
WHERE tenant_id = $1
ORDER BY updated_at DESC, id
LIMIT $2;

-- name: UpdateIncubationProject :one
UPDATE incubation_projects
SET status = $3, version = $4, updated_at = now()
WHERE tenant_id = $1 AND id = $2
RETURNING *;
