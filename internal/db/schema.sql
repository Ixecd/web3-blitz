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
    level      INTEGER NOT NULL DEFAULT 0,
    username   TEXT NOT NULL UNIQUE,
    password   TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id),
    token      TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS withdrawal_limits (
    id          BIGSERIAL PRIMARY KEY,
    level       INTEGER NOT NULL UNIQUE,
    level_name  TEXT NOT NULL,
    btc_daily   NUMERIC(20,8) NOT NULL,
    eth_daily   NUMERIC(20,8) NOT NULL,
    min_deposit NUMERIC(20,8) NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dead_letters (
    id         BIGSERIAL PRIMARY KEY,
    type       TEXT NOT NULL,
    payload    JSONB NOT NULL,
    error      TEXT NOT NULL,
    retries    INTEGER NOT NULL DEFAULT 0,
    resolved   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 角色表
CREATE TABLE IF NOT EXISTS roles (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,  -- admin / operator / user
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 权限表
CREATE TABLE IF NOT EXISTS permissions (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,  -- user:read / user:upgrade / limit:write 等
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 角色-权限关联
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       BIGINT NOT NULL REFERENCES roles(id),
    permission_id BIGINT NOT NULL REFERENCES permissions(id),
    PRIMARY KEY (role_id, permission_id)
);

-- 用户-角色关联
CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT NOT NULL REFERENCES users(id),
    role_id BIGINT NOT NULL REFERENCES roles(id),
    PRIMARY KEY (user_id, role_id)
);
