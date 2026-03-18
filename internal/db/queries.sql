-- name: CreateDeposit :exec
INSERT INTO deposits (tx_id, address, user_id, amount, height, confirmed, chain)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetDepositByTxID :one
SELECT * FROM deposits WHERE tx_id = ? LIMIT 1;

-- name: ListUnconfirmedDeposits :many
SELECT * FROM deposits 
WHERE confirmed = 0 
ORDER BY height ASC;

-- name: CreateDepositAddress :exec
INSERT OR IGNORE INTO deposit_addresses (user_id, address, chain, path)
VALUES (?, ?, ?, ?);

-- name: GetAddressByAddress :one
SELECT * FROM deposit_addresses WHERE address = ? LIMIT 1;

-- name: ListAddressesByUserID :many
SELECT * FROM deposit_addresses WHERE user_id = ? ORDER BY created_at ASC;

-- name: ListAllDepositAddresses :many
SELECT * FROM deposit_addresses ORDER BY created_at ASC;

-- name: ListDepositsByUserID :many
SELECT * FROM deposits WHERE user_id = ? ORDER BY created_at DESC;

-- name: GetTotalDepositByUserIDAndChain :one
SELECT COALESCE(SUM(amount), 0) as total
FROM deposits 
WHERE user_id = ? AND chain = ? AND confirmed = 1;

-- name: ListDepositsByChain :many
SELECT * FROM deposits WHERE chain = ? ORDER BY created_at DESC;

-- name: GetTotalDepositByChain :one
SELECT COALESCE(SUM(amount), 0) as total
FROM deposits
WHERE chain = ? AND confirmed = 1;

-- name: GetAllChainsTotalDeposit :one
SELECT COALESCE(SUM(amount), 0) as total
FROM deposits
WHERE confirmed = 1;

-- name: CreateWithdrawal :one
INSERT INTO withdrawals (address, user_id, amount, fee, status, chain)
VALUES (?, ?, ?, 0, 'pending', ?)
RETURNING *;

-- name: UpdateWithdrawalTx :exec
UPDATE withdrawals
SET tx_id = ?, fee = ?, status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ListWithdrawalsByUserID :many
SELECT * FROM withdrawals WHERE user_id = ? ORDER BY created_at DESC;