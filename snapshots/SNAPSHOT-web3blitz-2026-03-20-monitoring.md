# web3-blitz 快照 — 监控告警完整落地

> 归档时间：2026-03-20
> 里程碑：monitoring — prometheus + alertmanager(Telegram) + grafana dashboard 完整落地

---

## 本次新增

### `monitoring/` 目录（全新）

```
monitoring/
├── README.md
├── prometheus/
│   ├── prometheus.yml              # 4个 scrape job：prometheus/etcd/blitz-wallet/wallet-service
│   └── rules/
│       └── blitz.yml               # 6条业务告警规则
├── alertmanager/
│   └── alertmanager.yml.example    # Telegram 模板（bot_token 不提交，加入 .gitignore）
└── grafana/
    └── dashboards/
        └── blitz.json              # 9个 Panel，覆盖充值/提币/异常/服务健康四个维度
```

### Prometheus scrape jobs

| job | target | 说明 |
|-----|--------|------|
| prometheus | localhost:9090 | 自身 |
| etcd | localhost:2379 | 仅保留3个关键指标 |
| blitz-wallet | localhost:2112 | regtest miner |
| wallet-service | localhost:2113 | 主服务 |

### 告警规则（6条）

| 规则 | 级别 | 触发条件 |
|------|------|---------|
| WithdrawBroadcastFailed | critical | 5min内提币失败 > 0，持续1min |
| DeadLetterQueued | warning | 5min内死信队列新增 > 0，持续1min |
| ETHReorgDetected | warning | 10min内检测到reorg，立即触发（for: 0m）|
| LockAcquireFailSpike | warning | 5min内锁获取失败 > 10次，持续2min |
| WalletServiceDown | critical | wallet-service 不可达，持续1min |
| EtcdDown | critical | etcd 不可达，持续1min |

### Alertmanager — Telegram 路由

- `group_by: [alertname, service]`
- `group_wait: 30s` / `group_interval: 5m` / `repeat_interval: 12h`
- 消息含：状态 / 服务 / 描述 / 时间，`parse_mode: Markdown`

### Grafana dashboard（9个 Panel）

| Panel | 类型 | 指标 |
|-------|------|------|
| 服务健康 | stat | `up{job}` |
| 充值总笔数 | stat | `blitz_deposit_total` |
| 提币总笔数 | stat | `blitz_withdraw_total` |
| 异常快览 | stat | 失败/死信/reorg 累计 |
| 充值速率 | timeseries | `rate(blitz_deposit_total[5m])` |
| 提币速率 | timeseries | `rate(blitz_withdraw_total[5m])`，failed 红线 |
| 充值/提币金额趋势 | timeseries | `rate(blitz_{deposit,withdraw}_amount_total[5m])` |
| 死信队列 & 锁失败 | timeseries | `rate(blitz_{dead_letter,lock_acquire_fail}_total[5m])` |
| ETH Reorg | timeseries | `increase(blitz_reorg_total[5m])`，柱状图橙色 |
