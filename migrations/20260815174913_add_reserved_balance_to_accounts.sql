-- +goose Up

ALTER TABLE accounts
ADD COLUMN reserved_balance DECIMAL(19, 4) NOT NULL DEFAULT 0.0000
AFTER balance;

-- +goose Down

ALTER TABLE accounts
DROP COLUMN reserved_balance;