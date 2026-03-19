CREATE TABLE IF NOT EXISTS deposit_addresses (
    id         BIGSERIAL PRIMARY KEY,
    user_id    TEXT NOT NULL,
    address    TEXT NOT NULL UNIQUE,
    chain      TEXT NOT NULL CHECK(chain IN ('btc','eth')),
    path       TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deposits (
    id         BIGSERIAL PRIMARY KEY,
    tx_id      TEXT NOT NULL UNIQUE,
    address    TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    amount     NUMERIC(20,8) NOT NULL,
    height     BIGINT NOT NULL,
    confirmed  INTEGER NOT NULL DEFAULT 0,
    chain      TEXT NOT NULL CHECK(chain IN ('btc','eth')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS withdrawals (
    id         BIGSERIAL PRIMARY KEY,
    tx_id      TEXT,
    address    TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    amount     NUMERIC(20,8) NOT NULL,
    fee        NUMERIC(20,8) NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'pending',
    chain      TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL PRIMARY KEY,
    username   TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);