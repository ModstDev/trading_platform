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
SELECT *
FROM orders
WHERE id = ?;

-- name: ListOrdersByAccountID :many
SELECT *
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

-- name: FindMatchingSellOrder :one
SELECT *
FROM orders
WHERE instrument_id = ?
  AND side = 'SELL'
  AND status = 'PENDING'
  AND price <= ?
  AND quantity > filled_quantity
ORDER BY price ASC, created_at ASC
LIMIT 1;

-- name: FindMatchingBuyOrder :one
SELECT *
FROM orders
WHERE instrument_id = ?
  AND side = 'BUY'
  AND status = 'PENDING'
  AND price >= ?
  AND quantity > filled_quantity
ORDER BY price DESC, created_at ASC
LIMIT 1;

-- name: UpdateFilledQuantity :execresult
UPDATE orders
SET
    filled_quantity = filled_quantity + ?
WHERE id = ?
  AND status = 'PENDING'
  AND quantity - filled_quantity >= ?;