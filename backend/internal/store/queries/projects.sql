-- name: CreateProject :one
INSERT INTO projects (title, description, deadline, team_size, context, invite_code, owner_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetProject :one
SELECT * FROM projects WHERE id = $1;

-- name: GetProjectForUpdate :one
SELECT * FROM projects WHERE id = $1 FOR UPDATE;

-- name: GetProjectByCode :one
SELECT * FROM projects WHERE invite_code = $1;

-- name: GetProjectByCodeForUpdate :one
SELECT * FROM projects WHERE invite_code = $1 FOR UPDATE;

-- name: UpdateProjectDetails :one
UPDATE projects
SET title = $2, description = $3, deadline = $4, context = $5, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateInviteCode :one
UPDATE projects SET invite_code = $2, updated_at = now() WHERE id = $1 RETURNING *;

-- name: SetProjectOwner :exec
UPDATE projects SET owner_id = $2, updated_at = now() WHERE id = $1;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;

-- name: ListOwnedProjects :many
SELECT * FROM projects WHERE owner_id = $1 ORDER BY created_at;

-- name: AddMember :execrows
INSERT INTO project_members (project_id, user_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: IsMember :one
SELECT EXISTS (SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2);

-- name: CountMembers :one
SELECT count(*)::int FROM project_members WHERE project_id = $1;

-- name: RemoveMember :execrows
DELETE FROM project_members WHERE project_id = $1 AND user_id = $2;

-- name: EarliestOtherMember :one
SELECT u.id, u.full_name
FROM project_members m JOIN users u ON u.id = m.user_id
WHERE m.project_id = $1 AND m.user_id <> $2
ORDER BY m.joined_at, u.id
LIMIT 1;

-- name: ListMembers :many
SELECT m.project_id, u.id, u.full_name, u.email, m.joined_at,
       (pr.user_id IS NOT NULL)::bool AS has_profile,
       s.id AS seat_id, s.position AS seat_position, s.name AS seat_name, a.state AS assignment_state
FROM project_members m
JOIN users u ON u.id = m.user_id
LEFT JOIN profiles pr ON pr.user_id = u.id
LEFT JOIN role_assignments a ON a.project_id = m.project_id AND a.user_id = m.user_id
LEFT JOIN plan_seats s ON s.id = a.seat_id
WHERE m.project_id = ANY (sqlc.arg(project_ids)::uuid[])
ORDER BY m.project_id, m.joined_at, u.id;

-- name: ListProjectCards :many
SELECT p.id, p.title, p.context, p.deadline, p.team_size, p.plan_status, p.owner_id, p.created_at,
       (SELECT count(*) FROM project_members m WHERE m.project_id = p.id)::int AS member_count,
       (SELECT count(*) FROM project_members m LEFT JOIN profiles pr ON pr.user_id = m.user_id
         WHERE m.project_id = p.id AND pr.user_id IS NULL)::int AS missing_profiles,
       (SELECT count(*) FROM plan_seats s WHERE s.project_id = p.id)::int AS seat_count,
       (SELECT count(*) FROM role_assignments a
         WHERE a.project_id = p.id AND a.user_id IS NOT NULL AND a.state = 'accepted')::int AS accepted_count,
       COALESCE((SELECT sum(t.estimate) FROM tasks t WHERE t.project_id = p.id AND t.status = 'done'), 0)::float8 AS done_hours,
       COALESCE((SELECT sum(t.estimate) FROM tasks t WHERE t.project_id = p.id), 0)::float8 AS all_hours
FROM projects p
JOIN project_members me ON me.project_id = p.id AND me.user_id = sqlc.arg(user_id)
ORDER BY p.created_at DESC;

-- name: CountOpenTasks :one
SELECT count(*)::int FROM tasks WHERE project_id = $1 AND status <> 'done';
