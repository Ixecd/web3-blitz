# Wallet Core 设计文档

> 当前版本：v0.1.1 | 状态：开发中

---

## 概述

wallet-service 是 web3-blitz 的核心服务，负责：

- HD 钱包地址派生
- 充值监控（Deposit Watcher）
- 提币广播（BTC / ETH）
- 链上数据持久化
- 分布式协调（etcd）
- REST API 对外暴露

---

## 架构

```
HTTP Client
     │
     ▼
┌─────────────────────────────────────┐
│           wallet-service            │
│                                     │
│  /api/v1/address  ──────────────────────► HDWallet (BIP44)
│  /api/v1/balance  ──────────────────────► BTC/ETH RPC
│  /api/v1/withdraw ──────────────────────► etcd 分布式锁
│  /api/v1/withdrawals                │       │
│  /api/v1/deposits                   │       ▼
│  /api/v1/balance/total              │   余额校验 → 广播
│  /metrics         ──────────────────────► Prometheus
│                                     │
│  AddressRegistry(Mem)               │
│       ▲         │                   │
│       │         ▼                   │
│  DepositWatcher ────────────────────────► bitcoind (扫块)
│  ETHDepositWatcher ─────────────────────► geth (扫块)
│                                     │
│  ConfirmChecker ────────────────────────► etcd 选主
│       │                             │
│       ▼                             │
│  ConfigWatcher  ────────────────────────► etcd watch
│                                     │
│       │                             │
│       ▼                             │
│  PostgreSQL (blitz)                 │
└─────────────────────────────────────┘
```

---

## 核心组件

### HDWallet

基于 BIP44 标准派生地址，底层库 `go-bip32`。

- BTC 路径：`m/44'/0'/0'/0/<index>`
- ETH 路径：`m/44'/60'/0'/0/<index>`
- index 由 userID hash 生成，同一用户每次得到相同地址（幂等）

### AddressRegistry

内存中维护 `address → userID` 映射，线程安全（读写锁）。服务启动时从 DB 恢复，保证重启不丢状态。

### DepositWatcher / ETHDepositWatcher

定期扫新块，检测充值到账。

| 链 | 间隔 | 写入状态 |
|----|------|---------|
| BTC | 3s | confirmed=0 |
| ETH | 5s | confirmed=0 |

充值写入 DB 时 `confirmed=0`，由 ConfirmChecker 异步确认。

### ConfirmChecker

每 30 秒查询所有 `confirmed=0` 的充值记录，检查块高差是否达到阈值：

| 链 | 所需确认块数 |
|----|------------|
| BTC | 6 |
| ETH | 12 |

**etcd 选主**：多副本部署时只有 Leader 执行，防止重复确认。

### ConfigWatcher

监听 etcd `/blitz/config/` 前缀，配置变更时热更新 RPC 连接，无需重启服务。

### 分布式锁

提币前通过 etcd lease + txn CAS 获取用户级锁，防止并发重复提币。锁 key：`/blitz/lock/withdraw:{user_id}:{chain}`，TTL 30s。

---

## 提币流程

```
POST /api/v1/withdraw
    │
    ▼
参数校验（user_id / to_address / amount / chain）
    │
    ▼
获取 etcd 分布式锁
    ├─ 失败 → 429 请勿重复提交
    │
    ▼
余额校验（confirmed充值总额 - completed提币总额 >= amount）
    ├─ 不足 → 400 余额不足
    │
    ▼
写入 withdrawals 表（status=pending）
    │
    ▼
广播链上交易
    ├─ BTC：SendToAddress → GetTransaction 回填 fee
    └─ ETH：构建交易 → EIP-155 签名 → SendTransaction
    │
    ▼
更新 withdrawals 表（status=completed/failed，tx_id，fee）
    │
    ▼
释放 etcd 锁
    │
    ▼
返回响应
```

---

## API

### POST /api/v1/address

```json
// 请求
{"user_id": "user001", "chain": "btc"}

// 响应
{"address": "bcrt1q...", "path": "m/44'/0'/0'/0/2872464479", "user_id": "user001"}
```

### GET /api/v1/balance

```
GET /api/v1/balance?address=bcrt1q...&chain=btc
```

### POST /api/v1/withdraw

```json
// 请求
{"user_id": "user001", "to_address": "bcrt1q...", "amount": 0.05, "chain": "btc"}

// 响应（成功）
{"id": 1, "tx_id": "...", "fee": 0.00007976, "status": "completed", ...}

// 响应（失败）
// 429: 请勿重复提交
// 400: 余额不足: 可用 0.20000000，请求 0.30000000
```

### GET /api/v1/withdrawals

```
GET /api/v1/withdrawals?user_id=user001
```

### GET /api/v1/deposits

```
GET /api/v1/deposits?user_id=user001
```

### GET /api/v1/balance/total

```
GET /api/v1/balance/total?user_id=user001&chain=btc
```

---

## 数据库

PostgreSQL，三张表：

| 表名 | 说明 |
|------|------|
| deposit_addresses | 已生成的充值地址 |
| deposits | 充值记录（confirmed=0待确认，=1已确认）|
| withdrawals | 提币记录（pending/completed/failed）|

金额字段使用 `NUMERIC(20,8)`，避免浮点精度问题。

---

## 待实现

- [ ] IAM（JWT 中间件，堵住 /withdraw 无鉴权漏洞）
- [ ] ETH reorg 处理（链重组时回滚已确认充值）
- [ ] consumeDeposits 死信队列（重试失败后持久化待处理）
- [ ] 多签支持
- [ ] 主网参数切换（RegressionNetParams → MainNetParams）
