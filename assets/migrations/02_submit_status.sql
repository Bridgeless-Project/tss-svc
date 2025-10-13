-- +migrate Up

ALTER TABLE deposits
    ADD COLUMN tx_data TEXT;

ALTER TABLE deposits
    ADD COLUMN submitted BOOLEAN NOT NULL DEFAULT true;

-- +migrate Down

ALTER TABLE deposits
    DROP COLUMN tx_data;

ALTER TABLE deposits
    DROP COLUMN submitted;

