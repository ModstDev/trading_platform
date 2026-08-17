-- +goose Up
CREATE TABLE executions (
    id CHAR(36) NOT NULL PRIMARY KEY,
    order_id CHAR(36) NOT NULL,
    account_id CHAR(36) NOT NULL,
    instrument_id CHAR(36) NOT NULL,
    quantity DECIMAL(19, 4) NOT NULL,
    price DECIMAL(19, 4) NOT NULL,
    executed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_executions_order
        FOREIGN KEY (order_id) REFERENCES orders(id),

    CONSTRAINT fk_executions_account
        FOREIGN KEY (account_id) REFERENCES accounts(id),

    CONSTRAINT fk_executions_instrument
        FOREIGN KEY (instrument_id) REFERENCES instruments(id)
);

-- +goose Down

DROP TABLE executions;
