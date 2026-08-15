-- +goose Up
CREATE TABLE orders (
    id CHAR(36) NOT NULL PRIMARY KEY,
    account_id CHAR(36) NOT NULL,
    instrument_id CHAR(36) NOT NULL,

    side VARCHAR(10) NOT NULL,
    type VARCHAR(10) NOT NULL,

    quantity DECIMAL(19, 4) NOT NULL,
    price DECIMAL(19, 4) NULL,

    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_orders_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id)
        ON DELETE RESTRICT,

    CONSTRAINT fk_orders_instrument
        FOREIGN KEY (instrument_id)
        REFERENCES instruments(id)
        ON DELETE RESTRICT
);

-- +goose Down

DROP TABLE orders;
