-- name: CreateInstrument :exec
INSERT INTO instruments  (
    id,
    symbol,
    name,
    type,
    currency
)
VALUES (?, ?, ?, ?, ?);

-- name: GetInstrumentByID :one
SELECT
    id,
    symbol,
    name,
    type,
    currency,
    created_at
FROM instruments
WHERE id = ?;

-- name: GetInstrumentBySymbol :one
SELECT
    id,
    symbol,
    name,
    type,
    currency,
    created_at
FROM instruments
WHERE symbol = ?;

-- name: ListInstruments :many
SELECT
    id,
    symbol,
    name,
    type,
    currency,
    created_at
FROM instruments
ORDER BY symbol;