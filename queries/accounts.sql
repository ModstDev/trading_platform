-- name: CreateAccount :exec
INSERT INTO accounts (
    id,
    user_id,
    balance,
    reserved_balance,
    currency
)
VALUES (?, ?, ?, ?, ?);

-- name: GetAccountByUserID :one
SELECT
    id,
    user_id,
    balance,
    reserved_balance,
    currency,
    created_at
FROM accounts
WHERE user_id = ?;

-- name: GetAccountByID :one
SELECT
    id,
    user_id,
    balance,
    reserved_balance,
    currency,
    created_at
FROM accounts
WHERE id = ?;

-- name: ReserveFunds :execresult
UPDATE accounts
SET reserved_balance = reserved_balance + ?
WHERE id = ?
  AND balance - reserved_balance >= ?;

-- name: ReleaseFunds :execresult
UPDATE accounts
SET reserved_balance = reserved_balance - ?
WHERE id = ?
    AND reserved_balance >= ?;

-- name: SpendReservedFunds :execresult
UPDATE accounts
SET
    balance = balance - ?,
    reserved_balance = reserved_balance - ?
WHERE id = ?
    AND reserved_balance >= ?;

-- name: ReceiveFunds :exec
UPDATE accounts
SET balance = balance + ?
WHERE id = ?;