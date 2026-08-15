-- name: CreateUser :exec
INSERT INTO users (
    id,
    email,
    password_hash
)
VALUES (?, ?, ?);

-- name: GetUserByEmail :one
SELECT
    id,
    email,
    password_hash,
    created_at
FROM users
WHERE email = ?;

-- name: GetUserByID :one
SELECT
    id,
    email,
    password_hash,
    created_at
FROM users
WHERE id = ?;