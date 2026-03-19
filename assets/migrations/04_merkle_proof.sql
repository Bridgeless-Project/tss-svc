-- +migrate Up
ALTER TABLE deposits 
    ADD COLUMN merkle_proof TEXT;
-- +migrate Down

ALTER TABLE deposits
    DROP COLUMN merkle_proof;