-- +migrate Up

ALTER TABLE deposits
    ADD COLUMN is_swap BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN min_destination_amount TEXT,
    ADD COLUMN swap_deadline TEXT,
    ADD COLUMN final_receiver  VARCHAR(100);

-- +migrate Down

ALTER TABLE deposits
    DROP COLUMN is_swap,
    DROP COLUMN min_destination_amount,
    DROP COLUMN swap_deadline,
    DROP COLUMN final_receiver;

    