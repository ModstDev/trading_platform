-- name: CreateExecution :exec
INSERT INTO executions (
    id,
    order_id,
    account_id,
    instrument_id,
    quantity,
    price
)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ListExecutionsByAccountID :many
SELECT
    id,
    order_id,
    account_id,
    instrument_id,
    quantity,
    price,
    executed_at
FROM executions
WHERE account_id = ?
ORDER BY executed_at DESC;