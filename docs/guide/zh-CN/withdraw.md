# 提币流程指南

> 覆盖 BTC 和 ETH 的完整提币实现，包括设计思路、代码结构和本地验证步骤。

---

## 一、设计思路

提币（Withdrawal）的核心问题是：**如何保证链上广播和数据库记录的一致性**。

我们采用"先落库再广播"的策略：

```
收到提币请求
    │
    ▼
写入 withdrawals 表（status = pending）
    │
    ▼
调用链上 RPC 广播交易
    │
    ├─ 成功 → 更新 tx_id、fee、status = completed
    └─ 失败 → 更新 status = failed，返回错误
```

这样即使广播后服务崩溃，DB 里也有 pending 记录，后续可以对账或重试。

---

## 二、代码结构

```
internal/
├── api/
│   └── handler.go          # Withdraw handler，负责参数校验、DB 写入、调用链层
├── wallet/
│   ├── btc/
│   │   ├── btc.go          # BTCWallet，持有 rpc 客户端
│   │   └── withdraw.go     # BTC 提币实现（SendToAddress）
│   └── eth/
│       ├── eth.go          # ETHWallet，持有 rpc 客户端 + hotKey
│       └── withdraw.go     # ETH 提币实现（手动签名广播）
└── db/
    ├── queries.sql         # CreateWithdrawal / UpdateWithdrawalTx SQL
    └── queries.sql.go      # sqlc 生成，不要手动修改
```

---

## 三、BTC 提币

### 实现原理

BTC 提币直接委托给 `bitcoind` 的热钱包：

```go
// internal/wallet/btc/withdraw.go
func (w *BTCWallet) Withdraw(ctx context.Context, toAddress string, amount float64) (WithdrawResult, error) {
    addr, err := btcutil.DecodeAddress(toAddress, &chaincfg.RegressionNetParams)
    satoshis, _ := btcutil.NewAmount(amount)
    txHash, err := w.rpc.SendToAddress(addr, satoshis)
    return WithdrawResult{TxID: txHash.String(), Fee: 0}, nil
}
```

`SendToAddress` 会自动：

- 选择 UTXO（硬币选择）
- 估算 fee
- 签名交易
- 广播到网络

所以 BTC 提币不需要我们自己管理私钥，bitcoind 的 `blitz_wallet` 就是热钱包。

### 前置条件

`blitz_wallet` 必须有足够余额。regtest 下通过挖块获得：

```bash
bitcoin-cli -regtest createwallet "blitz_wallet"
bitcoin-cli -regtest -rpcwallet=blitz_wallet -generate 101
# coinbase 需要 101 个确认才能花费
```

---

## 四、ETH 提币

### 实现原理

ETH 没有 bitcoind 这样的托管钱包，需要我们自己持有私钥、手动构建并签名交易：

```go
// internal/wallet/eth/withdraw.go
func (w *ETHWallet) Withdraw(ctx context.Context, toAddress string, amount float64) (WithdrawResult, error) {
    // 1. 获取 nonce（防止重放）
    nonce, _ := w.rpc.PendingNonceAt(ctx, fromAddr)

    // 2. 获取当前 gasPrice
    gasPrice, _ := w.rpc.SuggestGasPrice(ctx)

    // 3. 构建交易（普通转账 gasLimit = 21000）
    tx := types.NewTransaction(nonce, toAddr, amountWei, 21000, gasPrice, nil)

    // 4. 用热钱包私钥签名（EIP-155，防止跨链重放）
    signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(chainID), w.hotKey)

    // 5. 广播
    w.rpc.SendTransaction(ctx, signedTx)
}
```

### 热钱包私钥配置

ETH 热钱包私钥通过环境变量注入，**不能写死在代码里**：

```bash
# 启动服务时传入
ETH_HOT_WALLET_KEY=<私钥hex> go run cmd/wallet-service/main.go
```

`main.go` 中读取并传入：

```go
hotKeyHex := os.Getenv("ETH_HOT_WALLET_KEY")
ethWallet := eth.NewETHWallet(hdWallet, ethRPC, registry, queries, hotKeyHex)
```

### 如何获取 geth dev 模式的热钱包私钥

geth `--dev` 模式启动时会预置一个账户，用以下步骤导出私钥：

```bash
# 1. 找 keystore 文件
docker exec -it <geth容器ID> find / -name "UTC--*" 2>/dev/null

# 2. 复制到宿主机
docker cp <容器ID>:"<keystore路径>" ./keystore.json

# 3. 用 Python 解密（dev 模式密码为空）
python3 -m venv /tmp/ethvenv && source /tmp/ethvenv/bin/activate
pip install eth-account

python3 - <<'EOF'
from eth_account import Account
import json

with open("./keystore.json") as f:
    keystore = json.load(f)

account = Account.from_key(Account.decrypt(keystore, ""))
print("私钥:", account.key.hex())
print("地址:", account.address)
EOF
```

---

## 五、数据库设计

```sql
CREATE TABLE IF NOT EXISTS withdrawals (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tx_id       TEXT,                          -- 广播后填入，pending 时为 NULL
    address     TEXT NOT NULL,                 -- 提币目标地址
    user_id     TEXT NOT NULL,
    amount      REAL NOT NULL,
    fee         REAL NOT NULL DEFAULT 0,       -- BTC 暂为 0，ETH 实时计算
    status      TEXT NOT NULL DEFAULT 'pending', -- pending / completed / failed
    chain       TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

关键 SQL：

```sql
-- 创建提币记录（初始 pending）
-- name: CreateWithdrawal :one
INSERT INTO withdrawals (address, user_id, amount, fee, status, chain)
VALUES (?, ?, ?, 0, 'pending', ?)
RETURNING *;

-- 广播后更新状态
-- name: UpdateWithdrawalTx :exec
UPDATE withdrawals
SET tx_id = ?, fee = ?, status = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?;
```

---

## 六、API

### POST /api/v1/withdraw

**请求：**

```json
{
  "user_id":    "test001",
  "to_address": "bcrt1q...",
  "amount":     0.05,
  "chain":      "btc"
}
```

**响应（成功）：**

```json
{
  "id":         2,
  "tx_id":      "c8d0ae077e7d22ff...",
  "user_id":    "test001",
  "to_address": "bcrt1q...",
  "amount":     0.05,
  "fee":        0,
  "status":     "completed",
  "chain":      "btc"
}
```

**响应（失败）：**

```
HTTP 500
提币失败: insufficient funds
```

---

## 七、本地验证

### BTC 提币完整流程

```bash
# 0. 启动依赖
docker compose up -d

# 1. 初始化热钱包（第一次需要）
bitcoin-cli -regtest createwallet "blitz_wallet"
bitcoin-cli -regtest -rpcwallet=blitz_wallet -generate 101

# 2. 启动服务
go run cmd/wallet-service/main.go

# 3. 生成充值地址
curl -X POST http://localhost:2113/api/v1/address \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test001","chain":"btc"}'
# → {"address":"bcrt1q..."}

# 4. 往充值地址打币（模拟用户充值）
bitcoin-cli -regtest -rpcwallet=blitz_wallet sendtoaddress "bcrt1q..." 0.5
bitcoin-cli -regtest -rpcwallet=blitz_wallet -generate 1
# 服务日志：💰 检测到充值！✅ deposit已写入DB

# 5. 生成提币目标地址
bitcoin-cli -regtest -rpcwallet=blitz_wallet getnewaddress

# 6. 发起提币
curl -X POST http://localhost:2113/api/v1/withdraw \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test001","to_address":"bcrt1q...","amount":0.05,"chain":"btc"}'
# → {"status":"completed","tx_id":"...","fee":0}
```

### ETH 提币完整流程

```bash
# 1. 获取热钱包私钥（见第四节）

# 2. 启动服务（注入私钥）
ETH_HOT_WALLET_KEY=<私钥hex> go run cmd/wallet-service/main.go

# 3. 生成两个 ETH 地址
curl -X POST http://localhost:2113/api/v1/address \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test001","chain":"eth"}'
# → {"address":"0xA87f..."}

curl -X POST http://localhost:2113/api/v1/address \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test002","chain":"eth"}'
# → {"address":"0xE5D7..."}

# 4. 发起 ETH 提币（从热钱包打到 test002）
curl -X POST http://localhost:2113/api/v1/withdraw \
  -H "Content-Type: application/json" \
  -d '{"user_id":"test001","to_address":"0xE5D7...","amount":0.01,"chain":"eth"}'
# → {"status":"completed","tx_id":"0xedd4...","fee":1.68e-13}

# 服务日志：
# 💰 ETH检测到充值! userID=test002 ...
# ✅ ETH deposit已写入DB
```

---

## 八、注意事项

**regtest vs 主网**

BTC 地址解析绑定了网络参数，regtest 用 `chaincfg.RegressionNetParams`，上主网需改为 `chaincfg.MainNetParams`，同时修改地址派生逻辑。

**ETH fee 精度**

`fee = gasLimit(21000) × gasPrice`，dev 模式 gasPrice 极低（约 1 wei），所以 fee 显示为 `1.68e-13` ETH，主网正常情况下约为 `0.0003` ETH。

**BTC fee 为 0 的原因**

`SendToAddress` 不直接返回实际扣除的 fee，需要后续通过 `GetTransaction(txid)` 回查 fee 字段回填，当前版本暂存 0，这是已知的 TODO。

**私钥安全**

生产环境私钥必须走 KMS 或 HSM，绝不能明文写在环境变量或配置文件中。当前 dev 模式的做法仅用于本地测试。