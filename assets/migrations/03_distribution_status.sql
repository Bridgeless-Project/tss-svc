-- +migrate Up

ALTER TABLE deposits
    ADD COLUMN distributed BOOLEAN NOT NULL DEFAULT true;

-- +migrate Down

ALTER TABLE deposits
    DROP COLUMN distributed;