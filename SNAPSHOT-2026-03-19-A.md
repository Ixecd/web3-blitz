# web3-blitz 项目快照

> 用途：新会话开始时直接把这个文件扔给 Claude，5秒对齐，继续工作。
> 最后更新：2026-03-19

---

## 项目是什么

Go + 云原生的交易所钱包充提币系统，目标是生产可用的完整交易所钱包基础设施。
脚手架工具：`github.com/Ixecd/dev-toolkit`（dtk），支持 1 行 init boilerplate + 1 键 AI-plan + Helm deploy。

---

## 当前版本：v0.1.1

### 已完成功能全览

**钱包核心：**
- BTC/ETH HD 钱包地址派生（BIP44，go-bip32）
- BTC Deposit Watcher（每3s扫块，regtest，6块确认）
- ETH Deposit Watcher（每5s扫块，geth dev，12块确认）
- ETH reorg 检测（ParentHash 校验，自动回退重扫，日志告警）
- AddressRegistry（内存映射 address→userID，读写锁，启动从DB恢复）
- BTC 提币（SendToAddress + GetTransaction fee 真实回填）
- ETH 提币（手动构建交易 + EIP-155 签名 + 广播）
- ConfirmChecker（每30s轮询，etcd 选主保证单活）

**风控体系：**
- 余额校验（confirmed充值总额 - completed提币总额 = 可用余额）
- 每日提币限额（4等级，滚动24小时窗口，从 JWT claims 取用户等级）
- etcd 分布式锁（防重复提币，lease TTL 30s，进程崩溃自动释放）

**用户与认证：**
- 用户注册/登录（bcrypt DefaultCost 密码加密）
- JWT 认证（access token 1h，HS256）
- Refresh token（随机32字节hex，7天，旋转策略，存DB可撤销）
- RBAC 权限系统（roles/permissions/role_permissions/user_roles 四张表）
- 权限中间件（JWTMiddleware + RBACMiddleware 链式组合）

**基础设施：**
- PostgreSQL + sqlc 持久化（11张表）
- etcd 分布式锁 + 选主 + 配置热更新
- Dead letter 死信队列（重试3次，1s/2s/3s递增，最终写DB持久化）
- 统一错误码系统（100xxx通用/101xxx用户/102xxx钱包）
- 统一响应格式（OK/Fail/FailMsg）
- Swagger 2.0 API 文档（docs/swagger.yaml）
- GitHub Actions CI（build + vet + test + docker build）
- Prometheus metrics + K8s Helm 部署

---

## 数据库表（11张）

| 表名 | 说明 | 关键字段 |
|------|------|---------|
| deposit_addresses | 充值地址 | user_id(TEXT), address, chain |
| deposits | 充值记录 | confirmed(0/1), height, amount(NUMERIC) |
| withdrawals | 提币记录 | status(pending/completed/failed), fee |
| users | 用户 | level(0-3), username, password(bcrypt) |
| refresh_tokens | refresh token | revoked, expires_at |
| withdrawal_limits | 等级限额配置 | level, btc_daily, eth_daily |
| dead_letters | 死信队列 | type, payload(JSONB), retries, resolved |
| roles | 角色 | name(admin/operator/user) |
| permissions | 权限点 | name(user:read/user:upgrade/limit:read/limit:write) |
| role_permissions | 角色-权限关联 | role_id, permission_id |
| user_roles | 用户-角色关联 | user_id, role_id |

---

## 提币限额等级

| 等级 | 名称 | BTC日限 | ETH日限 | 升级条件(累计BTC充值) |
|------|------|---------|---------|-------------------|
| 0 | 普通用户 | 2 | 50 | 默认 |
| 1 | 白银用户 | 10 | 200 | >= 1 BTC |
| 2 | 黄金用户 | 50 | 1000 | >= 10 BTC |
| 3 | 钻石用户 | 200 | 5000 | >= 50 BTC |

---

## RBAC 权限设计

| 角色 | 权限 | 说明 |
|------|------|------|
| admin | user:read, user:upgrade, limit:read, limit:write | 超级管理员 |
| operator | user:read, limit:read | 运营人员，只读 |
| user | 无特殊权限 | 普通用户 |

**分配管理员（SQL）：**
```sql
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r
WHERE u.username = 'alice' AND r.name = 'admin'
ON CONFLICT DO NOTHING;
```

---

## API 列表（完整）

| Method | Path | 认证 | 权限 | 说明 |
|--------|------|------|------|------|
| POST | /api/v1/register | 无 | 无 | 用户注册 |
| POST | /api/v1/login | 无 | 无 | 登录，返回 access_token + refresh_token |
| POST | /api/v1/refresh | 无 | 无 | 换新 token（旋转策略）|
| POST | /api/v1/logout | 无 | 无 | 撤销 refresh_token |
| GET  | /api/v1/users/me | JWT | 无 | 当前用户信息 |
| GET  | /api/v1/users | JWT | user:read | 用户列表 |
| POST | /api/v1/users/upgrade | JWT | user:upgrade | 升级用户等级 |
| GET  | /api/v1/withdrawal-limits | JWT | limit:read | 查看所有限额配置 |
| PUT  | /api/v1/withdrawal-limits/update | JWT | limit:write | 修改限额配置 |
| POST | /api/v1/address | 无 | 无 | 生成充值地址（幂等）|
| GET  | /api/v1/balance | 无 | 无 | 查询链上余额 |
| GET  | /api/v1/deposits | 无 | 无 | 查询充值记录 |
| GET  | /api/v1/balance/total | 无 | 无 | 查询累计已确认充值 |
| POST | /api/v1/withdraw | JWT | 无 | 发起提币（余额+限额+分布式锁）|
| GET  | /api/v1/withdrawals | 无 | 无 | 查询提币历史 |
| GET  | /metrics | 无 | 无 | Prometheus |

---

## 目录结构（完整）

```
internal/
├── api/
│   ├── handler.go       # 所有 HTTP handler（统一错误码+响应格式）
│   ├── server.go        # NewMux(h, jwtSecret, queries)，路由+中间件挂载
│   └── response.go      # OK() / Fail() / FailMsg()
├── auth/
│   ├── auth.go          # HashPassword, CheckPassword, GenerateToken, ParseToken, GenerateRefreshToken
│   ├── middleware.go    # JWTMiddleware(secret, next), GetClaims(r)
│   └── rbac.go          # RBACMiddleware(queries, permission, next), HasPermission
├── config/
│   ├── rpc.go           # BTCRPCHolder / ETHRPCHolder（sync.RWMutex 保护）
│   └── watcher.go       # ConfigWatcher，watch /blitz/config/ 前缀热更新
├── db/
│   ├── schema.sql       # 11张表结构
│   ├── seed.sql         # withdrawal_limits + roles + permissions 种子数据
│   ├── queries.sql      # 所有 SQL 查询（@param 风格）
│   ├── connect.go       # NewDB + runSeed（幂等初始化）
│   └── queries.sql.go   # sqlc 生成，不要手动修改
├── lock/
│   └── lock.go          # etcd 分布式锁（lease + txn CAS，TTL 30s）
├── pkg/
│   └── code/
│       ├── code.go          # 错误码定义（3段：100xxx/101xxx/102xxx）
│       └── code_generated.go # Message() + HTTPStatus()
├── wallet/
│   ├── btc/
│   │   ├── btc.go       # BTCWallet（GenerateDepositAddress, GetBalance）
│   │   ├── withdraw.go  # Withdraw + queryFee（GetTransaction回填）
│   │   └── watcher.go   # DepositWatcher，confirmed=false写入
│   ├── eth/
│   │   ├── eth.go       # ETHWallet（GenerateDepositAddress, GetBalance）
│   │   ├── withdraw.go  # Withdraw（EIP-155签名，PendingNonceAt防重放）
│   │   └── watcher.go   # ETHDepositWatcher + reorg检测（ParentHash校验）
│   ├── core/
│   │   ├── hd.go        # HDWallet（BIP44派生，go-bip32）
│   │   └── confirm.go   # ConfirmChecker + etcd选主（lease+keepalive）
│   └── types/           # 共享类型（Chain, AddressRegistry, DepositRecord）
cmd/
├── wallet-service/main.go  # 服务入口，consumeDeposits含重试+死信队列
├── chain-miner/main.go     # 双链挖矿+Prometheus（开发用）
└── pos-sim/main.go         # ETH PoS 收益模拟
docs/
├── swagger.yaml             # Swagger 2.0 完整 API 文档
├── design/
│   ├── wallet-core.md       # 充提币系统设计
│   └── etcd-architecture.md # etcd 三场景架构设计
└── guide/zh-CN/
    ├── quickstart.md
    ├── deploy.md
    ├── withdraw.md
    ├── auth.md
    ├── withdrawal-limits.md
    ├── ci.md
    ├── etcd.md
    └── postgresql-migration.md
.github/workflows/ci.yml
Procfile.single              # goreman 单节点 etcd
```

---

## 本地启动

```bash
# 1. 启动 etcd
goreman -f Procfile.single start

# 2. 启动依赖（bitcoind + geth + postgres）
docker compose up -d

# 3. BTC 初始化（第一次）
bitcoin-cli -regtest createwallet "blitz_wallet"
bitcoin-cli -regtest -rpcwallet=blitz_wallet -generate 101

# 4. 启动服务
DATABASE_URL=postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable \
ETH_HOT_WALLET_KEY=<私钥hex> \
go run cmd/wallet-service/main.go
```

---

## etcd key 总览

| key | 用途 | TTL |
|-----|------|-----|
| `/blitz/lock/withdraw:{uid}:{chain}` | 提币分布式锁 | 30s |
| `/blitz/leader/confirm-checker` | ConfirmChecker 选主 | 15s |
| `/blitz/config/btc_rpc_host` | BTC RPC 热更新 | 永久 |
| `/blitz/config/eth_rpc_host` | ETH RPC 热更新 | 永久 |

触发热更新：
```bash
etcdctl --endpoints=localhost:2379 put /blitz/config/btc_rpc_host "localhost:18443/wallet/blitz_wallet"
```

---

## Swagger

```bash
swagger validate docs/swagger.yaml
swagger serve docs/swagger.yaml
make swagger.validate
make swagger.serve
```

---

## 测试

```bash
# 充提币完整流程
./scripts/test_withdraw.sh

# 集成测试（无需外部服务）
go test ./test/integration/... -v

# e2e 测试（需服务启动）
go test -tags e2e ./test/e2e/... -v -timeout 120s
```

---

## 待实现（按优先级）

### 1. 前端（React + Vite + Tailwind + shadcn/ui）
管理后台：充值地址列表、提币审核、余额总览、用户等级管理、限额配置
技术栈：React + Vite + Tailwind + shadcn/ui + TanStack Query
注意：审美要求极高，留足时间

### 2. 监控告警体系
Grafana 完整仪表盘补全 + Alertmanager + 企业微信/钉钉/Telegram 告警

### 3. 主网切换 + 冷热钱包分离
BTC_NETWORK/ETH_RPC_HOST/WALLET_HD_SEED 走 etcd ConfigWatcher
BTC 2-of-3 多签，ETH Gnosis Safe
主网参数一键切换（regtest → testnet → mainnet）

### 4. 完整 IAM 补全
- 角色分配接口（POST /api/v1/users/:id/roles）
- 审计日志（谁在什么时候做了什么）
- 黑名单 token（access token 主动失效）
- IP 限流
- KYC 占位

### 5. user_id 统一
deposits/withdrawals.user_id TEXT → BIGINT
解锁 checkAndUpgradeLevel 自动升级等级

### 6. ETH reorg 数据库回滚
当前只检测+日志，补上回滚已确认充值记录

### 7. 死信队列完善
定时重试 + 人工介入管理接口（GET /api/v1/dead-letters）

### 8. 多链扩展（TRON / SOL / Polygon）

### 9. CI/CD 灰度发布（Argo Rollouts）

---

## 技术决策备忘

**数据：**
- deposits/withdrawals.user_id 是 TEXT（历史遗留），users.id 是 BIGINT，暂时并存
- JWT claims 里的 user_id 是 BIGINT，提币限额校验用 claims.UserID
- NUMERIC(20,8) 存金额，sqlc 生成 string，handler 层转 float64 返回

**认证：**
- JWT access token 1h，refresh token 7天旋转策略
- RBAC：JWT → RBACMiddleware → HasPermission → GetUserPermissions（JOIN查询）
- 角色分配目前只能 SQL，接口待补

**etcd：**
- 分布式锁：lease + txn CAS，崩溃自动释放，无死锁风险
- 选主：campaign loop + keepalive，TTL 15s内完成 failover
- 热更新：watch prefix，重建连接后原子 swap holder

**可靠性：**
- Dead letter：重试3次（1s/2s/3s），失败写 dead_letters 表，payload 存 JSONB
- ETH reorg：ParentHash 校验，不匹配则回退 lastHeight/lastHash，下轮重扫
- BTC reorg：6块确认天然防护

**代码规范：**
- 错误响应：`{"code": 102002, "message": "Insufficient balance"}`
- 成功响应：`{"data": {...}}`
- e2e/smoke 测试加 `//go:build e2e`，CI 只跑 integration

---

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| DATABASE_URL | PostgreSQL 连接串 | postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable |
| ETH_HOT_WALLET_KEY | ETH 热钱包私钥 hex | 空（提币不可用）|
| WALLET_HD_SEED | HD 钱包种子 | 空（使用测试 seed，生产必须设置）|
| JWT_SECRET | JWT 签名密钥 | dev-secret（生产必须设置，用 openssl rand -hex 32）|
| ETCD_ENDPOINTS | etcd 地址 | localhost:2379 |
| PORT | HTTP 服务端口 | 2113 |
| BTC_RPC_HOST | bitcoind RPC 地址 | localhost:18443/wallet/blitz_wallet |
| ETH_RPC_HOST | geth RPC 地址 | http://localhost:8545 |

---

## 历史快照

```
snapshots/
├── SNAPSHOT-2026-03-18-pg-migration.md      # PostgreSQL 迁移完成
├── SNAPSHOT-2026-03-18-etcd.md              # etcd 三场景完成
├── SNAPSHOT-2026-03-19-auth-limits.md       # 用户系统+JWT+refresh token+提币限额
├── SNAPSHOT-2026-03-19-final.md             # 错误码+死信+reorg+swagger
└── SNAPSHOT-2026-03-19-rbac.md             # RBAC+用户体系+限额管理后台
```
