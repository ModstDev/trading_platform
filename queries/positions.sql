-- name: GetPosition :one
SELECT
    id,
    account_id,
    instrument_id,
    quantity,
    average_price,
    created_at
FROM positions
WHERE account_id = ?
    AND instrument_id = ?;

-- name: CreatePosition :exec
INSERT INTO positions (
    id,
    account_id,
    instrument_id,
    quantity,
    average_price
)
VALUES (?, ?, ?, ?, ?);

-- name: UpdatePosition :exec
UPDATE positions
SET
    quantity = ?,
    average_price = ?
WHERE id = ?;

-- name: ListPositionsByAccountID :many
SELECT
    id,
    account_id,
    instrument_id,
    quantity,
    average_price,
    created_at
FROM positions
WHERE account_id = ?
  AND quantity > 0
ORDER BY created_at DESC;