-- name: CreateDeposit :exec
INSERT INTO deposits (tx_id, address, user_id, amount, height, confirmed, chain)
VALUES (@tx_id, @address, @user_id, @amount, @height, @confirmed, @chain);

-- name: GetDepositByTxID :one
SELECT * FROM deposits WHERE tx_id = @tx_id LIMIT 1;

-- name: ListUnconfirmedDeposits :many
SELECT * FROM deposits
WHERE confirmed = 0
ORDER BY height ASC;

-- name: CreateDepositAddress :exec
INSERT INTO deposit_addresses (user_id, address, chain, path)
VALUES (@user_id, @address, @chain, @path)
ON CONFLICT (address) DO NOTHING;

-- name: GetAddressByAddress :one
SELECT * FROM deposit_addresses WHERE address = @address LIMIT 1;

-- name: ListAddressesByUserID :many
SELECT * FROM deposit_addresses WHERE user_id = @user_id ORDER BY created_at ASC;

-- name: ListAllDepositAddresses :many
SELECT * FROM deposit_addresses ORDER BY created_at ASC;

-- name: ListDepositsByUserID :many
SELECT * FROM deposits WHERE user_id = @user_id ORDER BY created_at DESC;

-- name: GetTotalDepositByUserIDAndChain :one
SELECT COALESCE(SUM(amount), 0) as total
FROM deposits
WHERE user_id = @user_id AND chain = @chain AND confirmed = 1;

-- name: ListDepositsByChain :many
SELECT * FROM deposits WHERE chain = @chain ORDER BY created_at DESC;

-- name: GetTotalDepositByChain :one
SELECT COALESCE(SUM(amount), 0) as total
FROM deposits
WHERE chain = @chain AND confirmed = 1;

-- name: GetAllChainsTotalDeposit :one
SELECT COALESCE(SUM(amount), 0) as total
FROM deposits
WHERE confirmed = 1;

-- name: CreateWithdrawal :one
INSERT INTO withdrawals (address, user_id, amount, fee, status, chain)
VALUES (@address, @user_id, @amount, 0, 'pending', @chain)
RETURNING *;

-- name: UpdateWithdrawalTx :exec
UPDATE withdrawals
SET tx_id = @tx_id, fee = @fee, status = @status, updated_at = NOW()
WHERE id = @id;

-- name: ListWithdrawalsByUserID :many
SELECT * FROM withdrawals WHERE user_id = @user_id ORDER BY created_at DESC;

-- name: UpdateDepositConfirmed :exec
UPDATE deposits
SET confirmed = 1, updated_at = NOW()
WHERE id = @id;

-- name: GetTotalWithdrawalByUserIDAndChain :one
SELECT COALESCE(SUM(amount), 0) as total
FROM withdrawals
WHERE user_id = @user_id AND chain = @chain AND status = 'completed';

-- name: CreateUser :one
INSERT INTO users (username, password)
VALUES (@username, @password)
RETURNING *;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = @username LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = @id LIMIT 1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES (@user_id, @token, @expires_at)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = @token AND revoked = FALSE AND expires_at > NOW()
LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked = TRUE
WHERE token = @token;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens
SET revoked = TRUE
WHERE user_id = @user_id;

-- name: GetUserLevel :one
SELECT level FROM users WHERE id = @id LIMIT 1;

-- name: UpdateUserLevel :exec
UPDATE users SET level = @level WHERE id = @id;

-- name: GetWithdrawalLimit :one
SELECT * FROM withdrawal_limits WHERE level = @level LIMIT 1;

-- name: GetLast24hWithdrawalByUserAndChain :one
SELECT COALESCE(SUM(amount), 0) as total
FROM withdrawals
WHERE user_id = @user_id
  AND chain = @chain
  AND status = 'completed'
  AND created_at > NOW() - INTERVAL '24 hours';

-- name: CreateDeadLetter :exec
INSERT INTO dead_letters (type, payload, error, retries)
VALUES (@type, @payload, @error, @retries);

-- name: ListUnresolvedDeadLetters :many
SELECT * FROM dead_letters
WHERE resolved = FALSE
ORDER BY created_at ASC;

-- name: ResolveDeadLetter :exec
UPDATE dead_letters
SET resolved = TRUE, updated_at = NOW()
WHERE id = @id;

-- name: ListUsers :many
SELECT id, level, username, created_at, updated_at FROM users ORDER BY created_at DESC;

-- name: GetUserRoles :many
SELECT r.* FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = @user_id;

-- name: GetRolePermissions :many
SELECT p.* FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
WHERE rp.role_id = @role_id;

-- name: GetUserPermissions :many
SELECT DISTINCT p.name FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = @user_id;

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id)
VALUES (@user_id, @role_id)
ON CONFLICT DO NOTHING;

-- name: RemoveRoleFromUser :exec
DELETE FROM user_roles WHERE user_id = @user_id AND role_id = @role_id;

-- name: GetRoleByName :one
SELECT * FROM roles WHERE name = @name LIMIT 1;

-- name: ListWithdrawalLimits :many
SELECT * FROM withdrawal_limits ORDER BY level ASC;

-- name: UpdateWithdrawalLimit :exec
UPDATE withdrawal_limits
SET btc_daily = @btc_daily, eth_daily = @eth_daily
WHERE level = @level;