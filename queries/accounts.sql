-- name: CreateAccount :exec
INSERT INTO accounts (
    id,
    user_id,
    balance,
    currency
)
VALUES (?, ?, ?, ?);

-- name: GetAccountByUserID :one
SELECT
    id,
    user_id,
    balance,
    currency,
    created_at
FROM accounts
WHERE user_id = ?;