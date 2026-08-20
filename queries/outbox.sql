-- name: CreateOutboxEvent :exec
INSERT INTO outbox_events (
    id,
    aggregate_id,
    event_type,
    subject,
    payload
)
VALUES (?, ?, ?, ?, ?);

-- name: GetUnpublishedOutboxEvents :many
SELECT
    id,
    aggregate_id,
    event_type,
    subject,
    payload,
    created_at,
    published_at
FROM outbox_events
WHERE published_at IS NULL
ORDER BY created_at
LIMIT ?;

-- name: MarkOutboxEventPublished :exec
UPDATE outbox_events
SET published_at = CURRENT_TIMESTAMP
WHERE id = ?;