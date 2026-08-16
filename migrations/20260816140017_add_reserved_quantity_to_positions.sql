-- +goose Up

ALTER TABLE positions
ADD COLUMN reserved_quantity DECIMAL(19, 4) NOT NULL DEFAULT 0.0000
AFTER quantity;

-- +goose Down

ALTER TABLE positions
DROP COLUMN reserved_quantity;
