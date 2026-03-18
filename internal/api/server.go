package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/api/v1/address", h.GenerateAddress)
	mux.HandleFunc("/api/v1/balance", h.GetBalance)
	mux.HandleFunc("/api/v1/deposits", h.ListDeposits)
	mux.HandleFunc("/api/v1/balance/total", h.GetTotalBalance)
	mux.HandleFunc("/api/v1/withdraw", h.Withdraw)
	return mux
}
