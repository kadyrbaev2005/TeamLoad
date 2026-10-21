-- name: CreateUser :one
INSERT INTO users (email, password_hash, full_name, consent_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetProfile :one
SELECT * FROM profiles WHERE user_id = $1;

-- name: UpsertProfile :one
INSERT INTO profiles (user_id, strengths, weaknesses, tags, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (user_id) DO UPDATE
SET strengths = EXCLUDED.strengths, weaknesses = EXCLUDED.weaknesses, tags = EXCLUDED.tags, updated_at = now()
RETURNING *;

-- name: MarkUserSuggestionsOutdated :execrows
UPDATE role_assignments SET outdated = true
WHERE user_id = $1 AND source IN ('ai', 'ai_accepted') AND NOT outdated;
