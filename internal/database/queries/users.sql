-- name: GetUserByID :one
SELECT
    id,
    username,
    email,
    password_hash,
    first_name,
    last_name,
    is_active,
    is_verified,
    created_at,
    updated_at
FROM users
WHERE
    id = $1
    AND is_active = true;

-- name: GetUserByUsername :one
SELECT
    id,
    username,
    email,
    password_hash,
    first_name,
    last_name,
    is_active,
    is_verified,
    created_at,
    updated_at
FROM users
WHERE
    username = $1
    AND is_active = true;

-- name: GetUserByEmail :one
SELECT
    id,
    username,
    email,
    password_hash,
    first_name,
    last_name,
    is_active,
    is_verified,
    created_at,
    updated_at
FROM users
WHERE
    email = $1
    AND is_active = true;

-- name: CreateUser :one
INSERT INTO
    users (
        username,
        email,
        password_hash,
        first_name,
        last_name
    )
VALUES ($1, $2, $3, $4, $5)
RETURNING
    id,
    username,
    email,
    first_name,
    last_name,
    is_active,
    is_verified,
    created_at,
    updated_at;

-- name: UpdateUser :one
UPDATE users
SET
    username = $2,
    email = $3,
    first_name = $4,
    last_name = $5,
    updated_at = NOW()
WHERE
    id = $1
    AND is_active = true
RETURNING
    id,
    username,
    email,
    first_name,
    last_name,
    is_active,
    is_verified,
    created_at,
    updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password_hash = $2,
    updated_at = NOW()
WHERE
    id = $1
    AND is_active = true;

-- name: VerifyUser :exec
UPDATE users
SET
    is_verified = true,
    updated_at = NOW()
WHERE
    id = $1;

-- name: DeactivateUser :exec
UPDATE users SET is_active = false, updated_at = NOW() WHERE id = $1;

-- name: ListActiveUsers :many
SELECT
    id,
    username,
    email,
    first_name,
    last_name,
    is_active,
    is_verified,
    created_at,
    updated_at
FROM users
WHERE
    is_active = true
ORDER BY created_at DESC
LIMIT $1
OFFSET
    $2;

-- name: CountActiveUsers :one
SELECT COUNT(*) FROM users WHERE is_active = true;