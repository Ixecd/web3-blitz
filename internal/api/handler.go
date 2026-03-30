package api

import (
	"database/sql"

	"github.com/Ixecd/web3-blitz/internal/db"
	"github.com/Ixecd/web3-blitz/internal/email"
	"github.com/Ixecd/web3-blitz/internal/lock"
	"github.com/Ixecd/web3-blitz/internal/wallet/btc"
	"github.com/Ixecd/web3-blitz/internal/wallet/eth"
)

type Handler struct {
	btcWallet *btc.BTCWallet
	ethWallet *eth.ETHWallet
	queries   *db.Queries
	db        *sql.DB
	mailer    *email.Mailer
	locker    *lock.DistributedLock
	jwtSecret string
}

func NewHandler(btcWallet *btc.BTCWallet, ethWallet *eth.ETHWallet, queries *db.Queries, db *sql.DB, locker *lock.DistributedLock, jwtSecret string, mailer *email.Mailer) *Handler {
	return &Handler{
		btcWallet: btcWallet,
		ethWallet: ethWallet,
		queries:   queries,
		db:        db,
		locker:    locker,
		jwtSecret: jwtSecret,
		mailer:    mailer,
	}
}
