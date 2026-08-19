-- +goose Up

CREATE TABLE outbox_events (
    id CHAR(36) NOT NULL PRIMARY KEY,
    aggregate_id CHAR(36) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    payload TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP NULL,

    INDEX idx_outbox_unpublished (published_at, created_at)
);

-- +goose Down

DROP TABLE outbox_events;