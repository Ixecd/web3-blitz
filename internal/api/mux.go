package api

import (
	"net/http"

	"github.com/Ixecd/web3-blitz/internal/auth"
	"github.com/Ixecd/web3-blitz/internal/db"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewMux(h *Handler, jwtSecret string, queries *db.Queries) *http.ServeMux {
	mux := http.NewServeMux()

	// 公开接口
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/v1/register", h.Register)
	mux.HandleFunc("/api/v1/login", h.Login)
	mux.HandleFunc("/api/v1/address", h.GenerateAddress)
	mux.HandleFunc("/api/v1/balance", h.GetBalance)
	mux.HandleFunc("/api/v1/deposits", h.ListDeposits)
	mux.HandleFunc("/api/v1/balance/total", h.GetTotalBalance)
	mux.HandleFunc("/api/v1/withdrawals", h.ListWithdrawals)
	mux.HandleFunc("/api/v1/refresh", h.Refresh)
	mux.HandleFunc("/api/v1/logout", h.Logout)
	mux.HandleFunc("/api/v1/forgot-password", h.ForgotPassword)
	mux.HandleFunc("/api/v1/reset-password", h.ResetPassword)

	// 需要 JWT 保护的接口
	mux.HandleFunc("/api/v1/withdraw", auth.JWTMiddleware(jwtSecret, h.Withdraw))
	// JWT 保护
	mux.HandleFunc("/api/v1/users/me", auth.JWTMiddleware(jwtSecret, h.GetMe))
	// JWT + RBAC 保护
	mux.HandleFunc("/api/v1/users", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "user:read", h.ListUsers)))
	mux.HandleFunc("/api/v1/users/upgrade", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "user:upgrade", h.UpgradeUser)))
	mux.HandleFunc("/api/v1/withdrawal-limits", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "limit:read", h.ListWithdrawalLimits)))
	mux.HandleFunc("/api/v1/withdrawal-limits/update", auth.JWTMiddleware(jwtSecret,
		auth.RBACMiddleware(queries, "limit:write", h.UpdateWithdrawalLimit)))

	return mux
}
