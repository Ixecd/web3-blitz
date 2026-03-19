package api

import (
	"net/http"

	"github.com/Ixecd/web3-blitz/internal/auth"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewMux(h *Handler, jwtSecret string) *http.ServeMux {
	mux := http.NewServeMux()

	// 公开接口
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/api/v1/register", h.Register)
	mux.HandleFunc("/api/v1/login", h.Login)
	mux.HandleFunc("/api/v1/address", h.GenerateAddress)
	mux.HandleFunc("/api/v1/balance", h.GetBalance)
	mux.HandleFunc("/api/v1/deposits", h.ListDeposits)
	mux.HandleFunc("/api/v1/balance/total", h.GetTotalBalance)
	mux.HandleFunc("/api/v1/withdrawals", h.ListWithdrawals)

	// 需要 JWT 保护的接口
	mux.HandleFunc("/api/v1/withdraw", auth.JWTMiddleware(jwtSecret, h.Withdraw))

	return mux
}
