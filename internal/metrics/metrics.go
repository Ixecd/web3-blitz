package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// 充值相关
	DepositTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_deposit_total",
		Help: "Total number of deposits detected",
	}, []string{"chain", "status"}) // status: detected / confirmed

	DepositAmount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_deposit_amount_total",
		Help: "Total deposit amount",
	}, []string{"chain"})

	// 提币相关
	WithdrawTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_withdraw_total",
		Help: "Total number of withdrawals",
	}, []string{"chain", "status"}) // status: completed / failed

	WithdrawAmount = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_withdraw_amount_total",
		Help: "Total withdrawal amount",
	}, []string{"chain"})

	// 死信队列
	DeadLetterTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_dead_letter_total",
		Help: "Total number of dead letters",
	}, []string{"type"})

	// reorg 告警
	ReorgTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_reorg_total",
		Help: "Total number of chain reorganizations detected",
	}, []string{"chain"})

	// etcd 锁等待
	LockAcquireFailTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_lock_acquire_fail_total",
		Help: "Total number of distributed lock acquire failures",
	}, []string{"key"})

	// 对账
	ReconcileMissingDeposits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_reconcile_missing_deposits_total",
		Help: "Total number of deposits found on-chain but missing in DB",
	}, []string{"chain"})

	ReconcilePhantomDeposits = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_reconcile_phantom_deposits_total",
		Help: "Total number of deposits in DB but not found on-chain",
	}, []string{"chain"})

	ReconcileAmountMismatch = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_reconcile_amount_mismatch_total",
		Help: "Total number of deposits with amount mismatch between chain and DB",
	}, []string{"chain"})

	ReconcileErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_reconcile_errors_total",
		Help: "Total number of reconciliation errors",
	}, []string{"chain", "type"})

	ReconcileDuration = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "blitz_reconcile_duration_seconds",
		Help: "Duration of the last reconciliation run in seconds",
	}, []string{"chain"})

	// 热→冷归集
	SweepBTCTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_sweep_btc_total",
		Help: "Total number of BTC hot-to-cold sweeps",
	}, []string{"result"}) // success / skipped / error

	SweepETHTTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_sweep_eth_total",
		Help: "Total number of ETH hot-to-cold sweeps",
	}, []string{"result"})

	SweepAmountTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "blitz_sweep_amount_total",
		Help: "Total amount swept to cold wallet",
	}, []string{"chain"})
)

func Init() {
	prometheus.MustRegister(
		DepositTotal,
		DepositAmount,
		WithdrawTotal,
		WithdrawAmount,
		DeadLetterTotal,
		ReorgTotal,
		LockAcquireFailTotal,
		ReconcileMissingDeposits,
		ReconcilePhantomDeposits,
		ReconcileAmountMismatch,
		ReconcileErrorsTotal,
		ReconcileDuration,
		SweepBTCTotal,
		SweepETHTTotal,
		SweepAmountTotal,
	)
}
