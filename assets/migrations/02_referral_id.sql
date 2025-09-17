-- +migrate Up
CREATE DOMAIN uint16 AS integer
    CHECK (VALUE BETWEEN 0 AND 65535);


ALTER TABLE deposits
    ADD COLUMN referral_id uint16 NOT NULL;

-- +migrate Down

ALTER TABLE deposits
    DROP COLUMN referral_id;

DROP DOMAIN uint16;