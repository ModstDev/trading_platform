-- +goose Up

CREATE TABLE positions (
    id CHAR(36) NOT NULL PRIMARY KEY,
    account_id CHAR(36) NOT NULL,
    instrument_id CHAR(36) NOT NULL,
    quantity DECIMAL(19, 4) NOT NULL DEFAULT 0.0000,
    average_price DECIMAL(19, 4) NOT NULL DEFAULT 0.0000,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY account_instrument (account_id, instrument_id),

    CONSTRAINT fk_positions_account
        FOREIGN KEY (account_id)
        REFERENCES accounts(id),

    CONSTRAINT fk_positions_instrument
        FOREIGN KEY (instrument_id)
        REFERENCES instruments(id)
);

-- +goose Down

DROP TABLE positions;