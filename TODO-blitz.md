# TODO — blitz 路线图

> 充提币系统，从开发环境走向可交付。
> 按优先级排列，持续更新。当前：v0.1.11

---

## 🔴 P0 — 核心功能联调

### 充值流程
- [ ] Deposit 页面接真实 API
- [ ] 生成 BTC / ETH 充值地址展示
- [ ] 充值历史列表
- [ ] 到账确认逻辑验证（regtest 环境）

### 提币流程
- [ ] Withdraw 页面接真实 API
- [ ] 余额校验、限额校验联调
- [ ] 提币状态追踪（pending → completed / failed）

### Dashboard
- [ ] 接真实数据：余额、最近充值、最近提币
- [ ] 数据定时刷新

---

## 🟡 P1 — 完善

### 账户体系
- [ ] 忘记密码限流（同一邮箱 1 分钟内只能发一次）
- [ ] 注册邮箱验证码验证
- [ ] refresh token 自动续期（前端拦截 401 自动刷新）
- [ ] 登出时踢掉所有设备选项

### Admin
- [ ] 用户列表显示 email 而非 username
- [ ] 升级用户等级联调验证

### K8s
- [ ] etcd 旧 registry 数据清理

---

## 🟢 P2 — 收尾

- [ ] 扫码登录
- [ ] 所有页面跨主题验收
- [ ] 提币审核流程（pending → 人工审核 → 广播）
- [ ] 充值地址二维码生成
- [ ] 交易记录导出 CSV
- [ ] 管理后台操作日志
- [ ] 多币种扩展（USDT、TRX 等）
- [ ] Prometheus + Grafana 监控配置
- [ ] 用 dtk 部署 blitz 到本地 K8s，作为 dtk 推广 demo

---

## ✅ 已完成

### v0.1.12 — 迁移剥离 + 链上对账

- [x] Migration Job（pre-upgrade hook）：`cmd/migrate` 独立二进制，`SKIP_MIGRATIONS` 守卫，Helm hook Job
- [x] Reconciliation CronJob：`cmd/reconcile` 对账入口，每 5 分钟扫 BTC/ETH 链上 vs DB，报告 missing/phantom/amount_mismatch
- [x] `queries.sql` 新增 `ListDepositAddressesByChain` + `ListDepositsByChainAndHeightRange`
- [x] Prometheus 对账指标 5 个：`blitz_reconcile_*`
- [x] build/docker/migrate + build/docker/reconcile 双新 Dockerfile
- [x] deploy reconcile-cronjob 子 chart（CronJob + ServiceAccount）

### v0.1.14 — 热→冷钱包归集

- [x] `internal/wallet/sweep/sweeper.go` — 定时扫热钱包余额，扣除 keep 阈值后全量转入冷钱包
- [x] BTC: `GetBalance(*)` → `balance - keep - feeBuffer` → `SendToAddress` 归集
- [x] ETH: `BalanceAt` → `balance - keep - gasCost` → 签名广播归集
- [x] `cmd/sweep` 独立二进制，2 分钟超时，可选 PushGateway
- [x] 3 个 Prometheus 指标: `blitz_sweep_btc_total` / `blitz_sweep_eth_total` / `blitz_sweep_amount_total`
- [x] audit.go: `sweep.executed` / `sweep.failed` 审计事件
- [x] `build/docker/sweep/Dockerfile` + `deploy/blitz/sweep-cronjob/` Helm CronJob（每 15 分钟）

### v0.1.13 — 提币人工审核

- [x] 提币 WITHDRAWAL_AUTO_APPROVE 开关 — 默认关闭，提币写 pending 后返回，不广播
- [x] 管理员审核端点：GET /api/v1/admin/withdrawals/pending + POST approve / reject
- [x] withdraw:review RBAC 权限保护审核端点
- [x] audit.go: withdraw.approved / withdraw.rejected 审计事件
- [x] broadcastWithdrawal 复用 — auto-approve 和审核通过共用同一广播逻辑
- [x] 向后兼容：WITHDRAWAL_AUTO_APPROVE=true 恢复老行为

### v0.1.12 — 迁移剥离 + 链上对账

- [x] `internal/api/handler.go` 拆分为 auth.go / wallet.go / admin.go
- [x] etcd 从 Deployment + emptyDir 迁移到 StatefulSet + PVC（数据持久化）
- [x] 敏感环境变量迁移到 K8s Secret（DATABASE_URL / WALLET_HD_SEED / SMTP_PASS / JWT_SECRET）
- [x] 非敏感配置保留在 values.yaml env 块
- [x] `scripts/create-secret.sh` 幂等创建/更新 secret
- [x] SMTP 注入 K8s 验证，忘记密码全链路在 K8s 跑通 ✅
- [x] blitz controller chart 迁移到独立 chart
- [x] controller 自愈 e2e 验证（wallet-service 删除后 ~13s 恢复）

### v0.1.10 及之前

- [x] HD 钱包生成（BTC + ETH）
- [x] 充值地址生成 + 注册表恢复
- [x] BTC / ETH 充值扫块监听
- [x] 确认数检查（ConfirmChecker，etcd 选主）
- [x] 提币接口（分布式锁防重、余额校验、日限额）
- [x] 死信队列（deposit 写入失败兜底）
- [x] JWT 认证 + refresh token 轮转
- [x] RBAC 权限系统（roles / permissions / user_roles）
- [x] 管理后台基础（用户列表、等级升级、限额配置）
- [x] slog 结构化日志（JSON，LOG_LEVEL=debug）
- [x] register / login 改用 email
- [x] 忘记密码全链路（QQ 邮箱 SMTP）
- [x] 注册新账号（Login 页内联切换）
- [x] 记住我（localStorage / sessionStorage 切换）
- [x] golang-migrate + embed.FS（迁移文件打进二进制，启动自动执行）
- [x] 自包含 Helm chart（零外部依赖）
- [x] initContainers 启动顺序（wait-postgres + wait-etcd）
- [x] K8s 全链路部署跑通（5 个 pod 全部 Running）
- [x] /healthz 路由
- [x] pgx v5 search_path 问题修复

---

> 两个项目的交汇点：blitz 跑通之后就是 dtk 最好的推广 demo。
> 每完成一项，移到 ✅ 已完成，并更新 SNAPSHOT。
