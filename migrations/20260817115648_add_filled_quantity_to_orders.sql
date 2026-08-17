-- +goose Up

ALTER TABLE orders
ADD COLUMN filled_quantity DECIMAL(19,4) NOT NULL DEFAULT 0.0000
AFTER quantity;

-- +goose Down

ALTER TABLE orders
DROP COLUMN filled_quantity;