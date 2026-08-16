-- name: CreateOrder :exec
INSERT INTO orders (
    id,
    account_id,
    instrument_id,
    side,
    type,
    quantity,
    price,
    status
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetOrderByID :one
SELECT
    id,
    account_id,
    instrument_id,
    side,
    type,
    quantity,
    price,
    status,
    created_at
FROM orders
WHERE id = ?;

-- name: ListOrdersByAccountID :many
SELECT
    id,
    account_id,
    instrument_id,
    side,
    type,
    quantity,
    price,
    status,
    created_at
FROM orders
WHERE account_id = ?
ORDER BY created_at DESC;

-- name: CancelOrder :exec
UPDATE orders
SET status = 'CANCELED'
WHERE id = ?
    AND account_id = ?
    AND status = 'PENDING';

-- name: ExecuteOrder :execresult
UPDATE orders
SET status = 'EXECUTED'
WHERE id = ?
    AND account_id = ?
    AND status = 'PENDING';