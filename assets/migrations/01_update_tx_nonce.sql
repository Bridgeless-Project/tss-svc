
-- +migrate Up

ALTER TABLE deposits
    ALTER COLUMN tx_nonce TYPE BIGINT USING tx_nonce::BIGINT;

ALTER TABLE deposits ALTER COLUMN tx_nonce SET NOT NULL;

-- +migrate Down

ALTER TABLE deposits
    ALTER COLUMN tx_nonce TYPE INT;

ALTER TABLE deposits ALTER COLUMN tx_nonce SET NOT NULL;
