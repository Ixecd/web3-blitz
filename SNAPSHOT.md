# web3-blitz 项目快照

> 用途：新会话开始时直接把这个文件扔给 Claude，5秒对齐，继续工作。
> 最后更新：2026-03-18

---

## 项目是什么

Go + 云原生的交易所钱包充提币系统。
脚手架工具：`github.com/Ixecd/dev-toolkit`（dtk），支持 1 行 init boilerplate + 1 键 AI-plan + Helm deploy。

---

## 当前版本：v0.1.1

### 已完成

- BTC HD 钱包地址派生（BIP44，go-bip32）
- ETH HD 钱包地址派生（BIP44）
- BTC Deposit Watcher（每3s扫块，regtest）
- ETH Deposit Watcher（每5s扫块，geth dev）
- AddressRegistry（内存映射 address→userID，读写锁，启动从DB恢复）
- SQLite + sqlc 持久化（deposit_addresses / deposits / withdrawals 三张表）
- REST API（internal/api 包，Handler + NewMux）
- BTC 提币（SendToAddress，委托 bitcoind 热钱包）
- ETH 提币（手动构建交易 + EIP-155 签名 + 广播，热钱包私钥从环境变量注入）
- Prometheus metrics
- K8s 一键部署（dtk deploy + Helm）

### API 列表

| Method | Path | 说明 |
|--------|------|------|
| POST | /api/v1/address | 生成充值地址 |
| GET  | /api/v1/balance | 查询链上余额 |
| GET  | /api/v1/deposits | 查询充值记录（by user_id）|
| GET  | /api/v1/balance/total | 查询累计充值（by user_id + chain）|
| POST | /api/v1/withdraw | 发起提币 |
| GET  | /metrics | Prometheus |

---

## 目录结构（关键部分）

```
internal/
├── api/
│   ├── handler.go       # 所有 HTTP handler
│   └── server.go        # NewMux，路由注册
├── db/
│   ├── schema.sql
│   ├── queries.sql
│   └── queries.sql.go   # sqlc 生成
├── wallet/
│   ├── btc/
│   │   ├── btc.go       # BTCWallet（GenerateDepositAddress, GetBalance）
│   │   ├── withdraw.go  # BTC 提币
│   │   └── watcher.go   # BTC DepositWatcher
│   ├── eth/
│   │   ├── eth.go       # ETHWallet（GenerateDepositAddress, GetBalance）
│   │   ├── withdraw.go  # ETH 提币
│   │   └── watcher.go   # ETH DepositWatcher
│   ├── core/            # HDWallet（BIP44 派生）
│   ├── registry/        # AddressRegistry（已移到 types）
│   └── types/           # 共享类型（Chain, AddressRegistry, DepositRecord等）
cmd/
└── wallet-service/main.go
configs/
├── project.env          # REGISTRY_PREFIX / ARCH / VERSION
└── wallet/config.yaml
```

---

## 本地启动

```bash
# 依赖
docker compose up -d   # bitcoind(regtest) + geth(dev)

# BTC 初始化（第一次）
bitcoin-cli -regtest createwallet "blitz_wallet"
bitcoin-cli -regtest -rpcwallet=blitz_wallet -generate 101

# 启动服务
ETH_HOT_WALLET_KEY=<私钥hex> go run cmd/wallet-service/main.go
```

ETH 热钱包私钥获取方式（geth dev 模式）：
```bash
docker cp <geth容器ID>:"$(docker exec <容器ID> find / -name 'UTC--*' 2>/dev/null | head -1)" ./keystore.json
python3 -m venv /tmp/v && source /tmp/v/bin/activate && pip install eth-account
python3 -c "
from eth_account import Account; import json
ks = json.load(open('./keystore.json'))
acc = Account.from_key(Account.decrypt(ks, ''))
print(acc.key.hex())
"
```

---

## 待实现（按优先级）

### 1. 余额校验（下一个要做的）
提币前检查用户 DB 余额是否足够，需要先看 `GetTotalDepositByUserIDAndChain` 的返回类型。
位置：`internal/api/handler.go` Withdraw handler，`CreateWithdrawal` 之前加校验。
需要的文件：`internal/db/queries.sql.go`（GetTotalDepositByUserIDAndChain 函数签名）

### 2. 提币历史查询
`GET /api/v1/withdrawals?user_id=`
queries.sql 已有 `ListWithdrawalsByUserID`，只需加 handler + 注册路由。

### 3. 确认数逻辑
BTC 需要 6 个块确认，ETH 需要 12 个块确认，现在是 1 块就写 confirmed=true。
位置：btc/watcher.go 和 eth/watcher.go。

### 4. PostgreSQL 迁移 + Redis
替换 SQLite，Redis 用于防重复提币（分布式锁）。

---

## 技术决策备忘

- HD 钱包底层库用 `github.com/tyler-smith/go-bip32`（替换了有历史包袱的 btcutil/hdkeychain）
- BTC 地址：P2WPKH bech32，网络参数 RegressionNetParams（上主网改 MainNetParams）
- ETH 提币签名：EIP-155（含 chainID，防跨链重放）
- Nonce：从 PendingNonceAt 获取，防止 ETH 交易重放
- 提币策略：先落库（pending）再广播，广播失败更新 status=failed，保证可追溯
- BTC fee 当前存 0（bitcoind 不直接返回 fee，需后续 GetTransaction 回填，已知 TODO）
- ETH 热钱包私钥通过环境变量注入，K8s 生产环境走 Secret

---

## 常见问题

**BTC 提币报 insufficient funds**：blitz_wallet 余额不足，执行 `-generate 101` 补充。

**ETH 提币报"热钱包未配置"**：启动时未传 `ETH_HOT_WALLET_KEY` 环境变量。

**geth CrashLoopBackOff（K8s）**：deployment 里加 `enableServiceLinks: false`，防止 K8s 注入 `GETH_PORT` 环境变量干扰 geth 启动。
