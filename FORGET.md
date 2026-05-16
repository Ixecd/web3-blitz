# FORGET — Blitz 安全审计与修复追踪

> 日期：2026-05-16
> 审计范围：wallet-service 全部 API、认证、钱包、数据层
> 状态：严重/高危已修，中危修复中

---

## 一、严重（已修 ✅）

### 1.1 公开端点无认证 ✅
- mux.go: 5 个财务端点加 JWT 保护
- wallet.go: GenerateAddress / ListDeposits / GetTotalBalance / ListWithdrawals 改从 claims 取 user_id
- Withdraw handler: user_id 强制来自 JWT claims，不再信任请求体

### 1.2 登录无暴力破解防护 ✅
- auth.go Login: 失败时加 500ms 固定延时
- 邮箱不存在和密码错误统一延时，防枚举

### 1.3 密钥硬编码 ✅
- main.go: JWT_SECRET 未设置 → fatal 退出
- 检测不安全默认值和短密钥

---

## 二、高危（已修 ✅）

### 2.1 无全局限速 ✅
- 新增 ratelimit.go: 令牌桶算法
- 全局限速 120 req/min，登录/注册 6 req/min
- 按 IP 限流 + 5 分钟自动清理

### 2.2 提币锁绑定 JWT ✅
- Withdraw 不再从请求体读 user_id
- 锁 key 由 JWT claims 派生，无法伪造

### 2.3 提币单笔上限 ✅
- BTC 单笔 ≤ 1.0，ETH 单笔 ≤ 10.0

### 2.4 ETH 地址格式校验 ✅
- 提币时校验 common.IsHexAddress

---

## 三、中危（已修 ✅）

### 3.1 密码强度仅校验长度 >= 8 ✅
- auth.go Register + ResetPassword: 长度 >= 8 + 字母+数字双必须

### 3.2 缺失 HTTP 安全头 ✅
- headers.go: nosniff / X-Frame-Options / XSS / Referrer-Policy

### 3.3 错误消息暴露内部细节 ✅
- response.go: 新增 FailInternal — 向用户返回通用消息，真实错误打 slog

### 3.4 日志可能泄漏敏感数据 ✅
- 生产日志默认 info，不输出 debug 级别 email/PII

---

## 四、低危（已修 ✅）

### 4.1 Refresh token rotate 顺序修复 ✅
- auth.go Refresh: 先创建新 token 再撤销旧 token，防 DB 写入失败导致用户被锁

### 4.2 审计日志 ✅
- 新增 audit 包：JSON Lines 格式审计日志
- 提币 submitted/completed/failed 三事件全程留痕

### 4.3 ETH gas price 上限 ✅
- eth/withdraw.go: maxGasPrice = 200 gwei，超限拒绝并提示稍后重试

---

## 五、HD 种子硬加密（新增 ✅）

### 明文种子退役 ✅
- WALLET_HD_SEED 环境变量不再使用
- 改为 AES-256-GCM 加密文件 + argon2id 密钥派生
- 密码通过交互输入或 SEED_PASSPHRASE 环境变量提供

### 加密工具 ✅
- 新增 cmd/encrypt-seed: 种子文件加密生成
- encrypt-seed <种子hex> <输出路径> → 密码提示 → 0600 权限输出

---

## 六、Sealed Secrets 部署加密（新增 ✅）

### kp secret seal 已接入 ✅
- 6 个密钥（JWT_SECRET / DATABASE_URL / SMTP_USER / SMTP_PASS / ETH_HOT_WALLET_KEY）全部 kubeseal 加密
- 输出 configs/secrets/blitz-sealed.yaml，安全提交 Git
- 部署到集群后 sealed-secrets controller 自动解密为 K8s Secret → Pod env

### 密钥安全双层 ✅
```
开发机  encrypt-seed → AES-256-GCM 文件
CI/Git  kp secret seal → SealedSecret
集群    kubeseal 解封 → K8s Secret → Pod 挂载
```

---

## 七、极致补强 — 安全焊死（新增 ✅）

### 7.1 口令输入防内存抓取 ✅
- seed.go ReadPassphrase: 输入期间 Setrlimit(RLIMIT_CORE=0)
- 输入后清空字节缓冲区，不残留明文

### 7.2 SealedSecret 私钥物理隔离 ✅
- sealed-secrets controller 解密私钥独立离线冷储
- 不与业务集群混放，加一道物理边界
- 集群被完全攻破 → 攻击者仍然无法解密 SealedSecret

### 7.3 无感知种子轮换预案 ✅
- rotation.go: 静默派生新 HD 分支（m/44'/60'/0'/<gen>）
- 新分支激活 → 资产逐步迁移 → 旧分支可继续收款但新提币走新分支
- 主动规避长期持钥风险：种子暴露窗口不随使用时间累计
- 种子哈希可链上公开，供校验完整性

---


## 八、架构债务清偿 — 迁移剥离 + 链上对账（新增 ✅）

### 8.1 数据库迁移从主服务剥离 ✅
- **问题**: `runMigrations()` 绑在主二进制 `NewDB()`，多副本启动时并发迁移有脏状态风险
- **修复**:
  - `internal/db/connect.go` 增加 `SKIP_MIGRATIONS` 守卫，主服务 Deployment 设 `SKIP_MIGRATIONS=true`
  - `cmd/migrate` 独立二进制，纯 `golang-migrate/migrate/v4`，不依赖任何 Blitz internal 包
  - Helm `pre-upgrade,pre-install` hook Job，迁移在主服务启动前完成
  - 向后兼容：未设 `SKIP_MIGRATIONS` 时 `make dev` 行为不变

### 8.2 链上充值对账 CronJob ✅
- **问题**: BTC/ETH deposit watcher 只向前扫块，重启间隙、死信、链重组可能导致漏账，无反向校验
- **修复**:
  - `internal/wallet/reconcile/reconciler.go`: 逐块扫链 vs DB 记录，三类发现 — missing（链上有 DB 无）、phantom（DB 有链上无）、amount_mismatch
  - `cmd/reconcile` 独立二进制，4 分钟超时，可选 PushGateway 推送
  - Helm CronJob 每 5 分钟执行，`concurrencyPolicy: Forbid` 防重叠
  - Prometheus 指标：`blitz_reconcile_missing_deposits_total` / `blitz_reconcile_phantom_deposits_total` / `blitz_reconcile_amount_mismatch_total` / `blitz_reconcile_errors_total` / `blitz_reconcile_duration_seconds`

### 8.3 构建与部署 ✅
- `build/docker/migrate/Dockerfile` — alpine 基镜像，builder 编译 `cmd/migrate`，SQL 文件复制到 `/migrations`
- `build/docker/reconcile/Dockerfile` — alpine 基镜像，builder 编译 `cmd/reconcile`
- `deployments/blitz/reconcile-cronjob/` — 独立 Helm 子 chart
- 主镜像 `build/docker/blitz/Dockerfile` 已移除 `COPY migrations`，镜像体积更小 & 攻击面缩减

## 九、提币人工审核闸门（新增 ✅）

### 9.1 问题
- Withdraw handler 验证通过后直接广播交易，无人工审核环节
- 攻击者绕过量额校验或内部人恶意操作 → 热钱包被提空
- 这是上线前的硬前置条件

### 9.2 修复
- `WITHDRAWAL_AUTO_APPROVE` 环境变量 — 默认关闭（人工审核），设为 `true` 恢复老行为
- 默认模式: Withdraw → CreateWithdrawal(pending) → 返回 `pending_review` → 管理员审核 → approve/reject
- 三个管理员端点: `GET /api/v1/admin/withdrawals/pending` / `POST approve` / `POST reject`
- `withdraw:review` RBAC 权限保护，审计日志 `withdraw.approved` / `withdraw.rejected` 全留痕
- `broadcastWithdrawal()` 复用 auto-approve 和审核通过两种路径
- SQL: `ListPendingWithdrawals` + `AdminApproveRejectWithdrawal`
- 错误码: `ErrWalletPendingReview` (202) / `ErrWalletInvalidStatus` (400)

### 9.3 状态流转
```
pending (用户提交)
├── admin approve → broadcasting → completed (广播成功)
│                                 └── failed (广播失败，DB 仍可重审)
└── admin reject  → rejected
```

## 十、热→冷钱包定时归集（新增 ✅）

### 10.1 问题
- `ETH_HOT_WALLET_KEY` 泄露 → 热钱包所有资金可被提空
- 热钱包长期累积用户充值，暴露窗口随金额和时间无限放大
- 没有自动减损机制

### 10.2 修复
- `internal/wallet/sweep/sweeper.go`: 每次归集扫描热钱包余额，扣除 keep 阈值后全量转入冷地址
- **BTC 归集**: `GetBalance("*")` 获取钱包总余额 → `balance - keep - feeBuffer` → `SendToAddress` 广播
- **ETH 归集**: `BalanceAt` 获取余额 → `balance - keep - gasCost` → EIP-155 签名 → `SendTransaction` 广播
- `cmd/sweep` 独立二进制，2 分钟超时，可选 PushGateway 推送
- Helm CronJob 每 15 分钟执行，`concurrencyPolicy: Forbid` 防重叠
- 冷钱包地址通过 `COLD_WALLET_BTC` / `COLD_WALLET_ETH` env 注入（K8s Secret）
- 保留金额可配置: `HOT_WALLET_KEEP_BTC` (默认 0.01), `HOT_WALLET_KEEP_ETH` (默认 0.1)
- BTC 手续费缓冲: `SWEEP_BTC_FEE_BUFFER` (默认 0.0001)

### 10.3 减损模型
```
热钱包风险敞口 = max(keepThreshold, 15 分钟内新增充值)
归集频率 15 分钟 → 最大损失窗口 < 15 分钟的充值流入
冷钱包私钥离线 → 即使热密钥泄露，攻击者最多拿走 keep + 最近 15 分钟充值
```

### 10.4 指标与审计
- `blitz_sweep_btc_total{result}` — success / skipped / error
- `blitz_sweep_eth_total{result}` — success / skipped / error
- `blitz_sweep_amount_total{chain}` — 累计归集金额
- `sweep.executed` / `sweep.failed` 审计日志

---
*全部修完，FORGET 清空。*


