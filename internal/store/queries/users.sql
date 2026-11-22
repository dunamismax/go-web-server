-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: RegisterUser :one
-- Self-service sign-up. The first account in an empty database becomes the
-- admin so a fresh deployment is never left without one.
INSERT INTO users (email, name, password_hash, role)
VALUES (
    sqlc.arg(email),
    sqlc.arg(name),
    sqlc.arg(password_hash),
    CASE WHEN EXISTS (SELECT 1 FROM users) THEN 'member' ELSE 'admin' END
)
RETURNING *;

-- name: CreateUser :one
INSERT INTO users (email, name, password_hash, role)
VALUES (sqlc.arg(email), sqlc.arg(name), sqlc.arg(password_hash), sqlc.arg(role))
RETURNING *;

-- name: ListUsers :many
SELECT * FROM users
WHERE sqlc.arg(search)::text = ''
   OR strpos(lower(name), lower(sqlc.arg(search)::text)) > 0
   OR strpos(email, lower(sqlc.arg(search)::text)) > 0
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: CountUsers :one
SELECT count(*) FROM users
WHERE sqlc.arg(search)::text = ''
   OR strpos(lower(name), lower(sqlc.arg(search)::text)) > 0
   OR strpos(email, lower(sqlc.arg(search)::text)) > 0;

-- name: UpdateUserProfile :one
UPDATE users
SET name = sqlc.arg(name), email = sqlc.arg(email), updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name = sqlc.arg(name),
    email = sqlc.arg(email),
    role = sqlc.arg(role),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = sqlc.arg(password_hash),
    password_changed_at = now(),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: UpgradeUserPasswordHash :exec
-- Re-hashes with current parameters after a successful login. The password
-- itself is unchanged, so password_changed_at stays put and sessions survive.
UPDATE users
SET password_hash = sqlc.arg(password_hash)
WHERE id = sqlc.arg(id);

-- name: SetUserActive :one
UPDATE users
SET is_active = sqlc.arg(is_active), updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;
