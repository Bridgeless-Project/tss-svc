-- +migrate Up

CREATE TABLE deposits
(
    id                  BIGSERIAL PRIMARY KEY,

    tx_hash             VARCHAR(100) NOT NULL,
    tx_event_id         INT          NOT NULL,
    chain_id            VARCHAR(50)  NOT NULL,

    depositor           VARCHAR(100),
    receiver            VARCHAR(100),
    deposit_amount      TEXT,
    withdrawal_amount   TEXT,
    deposit_token       VARCHAR(100),
    withdrawal_token    VARCHAR(100),
    is_wrapped_token    BOOLEAN DEFAULT false,
    deposit_block       BIGINT,
    signature           TEXT,

    withdrawal_status   int          NOT NULL,

    withdrawal_tx_hash  VARCHAR(100),
    withdrawal_chain_id VARCHAR(50),

    CONSTRAINT unique_deposit UNIQUE (tx_hash, tx_event_id, chain_id)
);

-- +migrate Down

DROP TABLE deposits;