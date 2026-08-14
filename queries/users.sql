-- name: CreateUser :execresult
INSERT INTO users (
    email,
    password_hash
)
VALUES (?, ?);

-- name: GetUserByEmail :one
SELECT
    id,
    email,
    password_hash,
    created_at,
    updated_at
FROM users
WHERE email = ?;