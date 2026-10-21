-- name: ListAssignments :many
SELECT * FROM role_assignments WHERE project_id = $1;

-- name: CountAssignments :one
SELECT count(*)::int FROM role_assignments WHERE project_id = $1;

-- name: DeleteAssignments :exec
DELETE FROM role_assignments WHERE project_id = $1;

-- name: InsertAssignment :exec
INSERT INTO role_assignments (project_id, seat_id, user_id, fit, runner_up_seat_id, explanation, state, source, prompt_version)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: AcceptAssignment :execrows
UPDATE role_assignments
SET state = 'accepted', source = CASE WHEN source = 'manual' THEN 'manual' ELSE 'ai_accepted' END
WHERE project_id = $1 AND seat_id = $2 AND user_id IS NOT NULL;

-- name: AcceptAllAssignments :execrows
UPDATE role_assignments
SET state = 'accepted', source = CASE WHEN source = 'manual' THEN 'manual' ELSE 'ai_accepted' END
WHERE project_id = $1 AND user_id IS NOT NULL AND state <> 'accepted';

-- name: GetAssignmentBySeat :one
SELECT * FROM role_assignments WHERE project_id = $1 AND seat_id = $2;

-- name: GetAssignmentByUser :one
SELECT * FROM role_assignments WHERE project_id = $1 AND user_id = $2;

-- name: ClearAssignmentUser :exec
UPDATE role_assignments SET user_id = NULL WHERE project_id = $1 AND seat_id = $2;

-- name: SetManualAssignment :exec
INSERT INTO role_assignments (project_id, seat_id, user_id, fit, runner_up_seat_id, explanation, state, source, outdated)
VALUES ($1, $2, $3, NULL, NULL, '', 'accepted', 'manual', false)
ON CONFLICT (project_id, seat_id) DO UPDATE
SET user_id = EXCLUDED.user_id, fit = NULL, runner_up_seat_id = NULL, explanation = '',
    state = 'accepted', source = 'manual', outdated = false, generated_at = now();

-- name: SetMatchingMeta :exec
UPDATE projects
SET matching_ran_at = now(), matching_source = $2, matching_prompt_version = $3,
    matching_gaps = $4, matching_growth_note = $5
WHERE id = $1;
