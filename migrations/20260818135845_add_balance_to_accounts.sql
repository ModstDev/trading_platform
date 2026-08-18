-- +goose Up
ALTER TABLE orders
ADD COLUMN max_cost DECIMAL(20,4) NULL;

-- +goose Down
ALTER TABLE orders
DROP COLUMN max_cost;
