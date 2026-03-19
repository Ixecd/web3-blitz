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