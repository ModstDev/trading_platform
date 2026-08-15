-- +goose Up

CREATE TABLE instruments (
    id CHAR(36) NOT NULL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,
    currency CHAR(3) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down

DROP TABLE instruments;