-- name: ListTasks :many
SELECT * FROM tasks WHERE project_id = $1 ORDER BY created_at, id;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = $1 AND project_id = $2;

-- name: CreateTask :one
INSERT INTO tasks (project_id, title, assignee_id, estimate, status, raci_activity_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateTask :one
UPDATE tasks
SET title = $3, assignee_id = $4, estimate = $5, status = $6, raci_activity_id = $7, updated_at = now()
WHERE id = $1 AND project_id = $2
RETURNING *;

-- name: DeleteTask :execrows
DELETE FROM tasks WHERE id = $1 AND project_id = $2;

-- name: RaciActivityInProject :one
SELECT EXISTS (SELECT 1 FROM raci_activities WHERE id = $1 AND project_id = $2);

-- name: InsertAICall :exec
INSERT INTO ai_calls (project_id, kind, provider, prompt_version, attempt, ok, latency_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: CountAICallsSince :one
SELECT count(*)::int FROM ai_calls WHERE project_id = $1 AND created_at >= $2;
