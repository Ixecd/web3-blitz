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
    email      TEXT NOT NULL UNIQUE,
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

CREATE TABLE IF NOT EXISTS roles (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       BIGINT NOT NULL REFERENCES roles(id),
    permission_id BIGINT NOT NULL REFERENCES permissions(id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT NOT NULL REFERENCES users(id),
    role_id BIGINT NOT NULL REFERENCES roles(id),
    PRIMARY KEY (user_id, role_id)
);

-- 初始化角色
INSERT INTO roles (name, description) VALUES
('admin',    '管理员'),
('operator', '运营'),
('user',     '普通用户')
ON CONFLICT (name) DO NOTHING;

-- 初始化权限
INSERT INTO permissions (name, description) VALUES
('user:read',    '查看用户列表'),
('user:upgrade', '升级用户等级'),
('limit:read',   '查看提币限额'),
('limit:write',  '修改提币限额')
ON CONFLICT (name) DO NOTHING;

-- admin 角色绑定所有权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- 初始化提币限额
INSERT INTO withdrawal_limits (level, level_name, btc_daily, eth_daily, min_deposit) VALUES
(0, '普通用户', '2.00000000',   '50.00000000',   '0.00000000'),
(1, '白银用户', '10.00000000',  '200.00000000',  '1.00000000'),
(2, '黄金用户', '50.00000000',  '1000.00000000', '10.00000000'),
(3, '钻石用户', '200.00000000', '5000.00000000', '50.00000000')
ON CONFLICT (level) DO NOTHING;
