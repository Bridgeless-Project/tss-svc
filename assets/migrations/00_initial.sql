-- +migrate Up

CREATE TABLE deposits
(
    id                  BIGSERIAL PRIMARY KEY,

    tx_hash             VARCHAR(100) NOT NULL,
    tx_nonce        BIGINT          NOT NULL,
    chain_id            VARCHAR(50)  NOT NULL,

    depositor           VARCHAR(100),
    receiver            VARCHAR(100) NOT NULL,
    deposit_amount      TEXT        NOT NULL,
    withdrawal_amount   TEXT       NOT NULL,
    commission_amount TEXT NOT NULL,
    deposit_token       VARCHAR(100) NOT NULL,
    withdrawal_token    VARCHAR(100) NOT NULL,
    is_wrapped_token    BOOLEAN DEFAULT false,
    deposit_block       BIGINT      NOT NULL,
    signature           TEXT,

    withdrawal_status   int          NOT NULL,

    withdrawal_tx_hash  VARCHAR(100),
    withdrawal_chain_id VARCHAR(50),

    CONSTRAINT unique_deposit UNIQUE (tx_hash, tx_nonce, chain_id)
);

-- +migrate Down

DROP TABLE deposits;