-- name: ListSeats :many
SELECT * FROM plan_seats WHERE project_id = $1 ORDER BY position;

-- name: DeleteSeats :exec
DELETE FROM plan_seats WHERE project_id = $1;

-- name: InsertSeat :one
INSERT INTO plan_seats (project_id, position, name, summary, responsibilities, skills, effort_share, why, color_token)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateSeat :execrows
UPDATE plan_seats SET name = $3, summary = $4, effort_share = $5
WHERE id = $1 AND project_id = $2;

-- name: SetPlanGenerated :exec
UPDATE projects
SET plan_status = 'draft', plan_source = $2, plan_prompt_version = $3, plan_generated_at = now(), updated_at = now()
WHERE id = $1;

-- name: SetPlanStatus :exec
UPDATE projects SET plan_status = $2, updated_at = now() WHERE id = $1;

-- name: ClearMatchingMeta :exec
UPDATE projects
SET matching_ran_at = NULL, matching_source = NULL, matching_prompt_version = NULL,
    matching_gaps = '[]', matching_growth_note = NULL
WHERE id = $1;

-- name: ClearRaciMeta :exec
UPDATE projects SET raci_source = NULL, raci_prompt_version = NULL, raci_generated_at = NULL WHERE id = $1;
