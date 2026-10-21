-- name: ListRaciActivities :many
SELECT * FROM raci_activities WHERE project_id = $1 ORDER BY position, id;

-- name: ListRaciCells :many
SELECT c.* FROM raci_cells c
JOIN raci_activities a ON a.id = c.activity_id
WHERE a.project_id = $1;

-- name: DeleteRaciActivities :exec
DELETE FROM raci_activities WHERE project_id = $1;

-- name: InsertRaciActivity :one
INSERT INTO raci_activities (project_id, position, name) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateRaciActivity :execrows
UPDATE raci_activities SET position = $3, name = $4 WHERE id = $1 AND project_id = $2;

-- name: DeleteRaciActivity :exec
DELETE FROM raci_activities WHERE id = $1 AND project_id = $2;

-- name: DeleteRaciCells :exec
DELETE FROM raci_cells WHERE activity_id = $1;

-- name: InsertRaciCell :exec
INSERT INTO raci_cells (activity_id, seat_id, value) VALUES ($1, $2, $3);

-- name: SetRaciMeta :exec
UPDATE projects SET raci_source = $2, raci_prompt_version = $3, raci_generated_at = now() WHERE id = $1;
