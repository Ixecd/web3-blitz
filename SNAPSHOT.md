# web3-blitz 项目快照

> 用途：新会话开始时直接把这个文件扔给 Claude，5秒对齐，继续工作。
> 最后更新：2026-03-20

---

## 项目是什么

Go + 云原生的交易所钱包充提币系统，目标是生产可用的完整交易所钱包基础设施。
脚手架工具：`github.com/Ixecd/dev-toolkit`（dtk），支持 1 行 init boilerplate + 1 键 AI-plan + Helm deploy。

---

## 当前版本：v0.1.2

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

**可靠性：**
- Dead letter 死信队列（重试3次 1s/2s/3s，最终写DB持久化，payload JSONB）
- 统一错误码系统（100xxx通用/101xxx用户/102xxx钱包）
- 统一响应格式（OK/Fail/FailMsg）

**监控（v0.1.2 完整落地）：**
- Prometheus 自定义业务指标（internal/metrics/metrics.go）
  - blitz_deposit_total（chain, status: detected/confirmed）
  - blitz_deposit_amount_total（chain）
  - blitz_withdraw_total（chain, status: completed/failed）
  - blitz_withdraw_amount_total（chain）
  - blitz_dead_letter_total（type）
  - blitz_reorg_total（chain）
  - blitz_lock_acquire_fail_total（key）
- prometheus.yml：4个 scrape job（prometheus/etcd/blitz-wallet/wallet-service）
- 告警规则 6条（monitoring/prometheus/rules/blitz.yml）
- Alertmanager Telegram 告警（routing + 12h repeat_interval）
- Grafana dashboard（9个 Panel，monitoring/grafana/dashboards/blitz.json）
- Swagger 2.0 API 文档（docs/swagger.yaml）
- GitHub Actions CI（build + vet + test + docker build）

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

| 等级 | 名称 | BTC日限 | ETH日限 |
|------|------|---------|---------|
| 0 | 普通用户 | 2 | 50 |
| 1 | 白银用户 | 10 | 200 |
| 2 | 黄金用户 | 50 | 1000 |
| 3 | 钻石用户 | 200 | 5000 |

---

## RBAC 权限设计

| 角色 | user:read | user:upgrade | limit:read | limit:write |
|------|-----------|--------------|------------|-------------|
| admin | ✅ | ✅ | ✅ | ✅ |
| operator | ✅ | ❌ | ✅ | ❌ |
| user | ❌ | ❌ | ❌ | ❌ |

分配管理员：
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

## 目录结构（关键部分）

```
internal/
├── api/handler.go, server.go, response.go
├── auth/auth.go, middleware.go, rbac.go
├── config/rpc.go, watcher.go
├── db/schema.sql, seed.sql, queries.sql, connect.go
├── lock/lock.go
├── metrics/metrics.go
├── pkg/code/code.go, code_generated.go
├── wallet/btc/, eth/, core/, types/
cmd/wallet-service/main.go
docs/
├── swagger.yaml
├── design/wallet-core.md, etcd-architecture.md, rbac.md, error-code.md, dead-letter.md
└── guide/zh-CN/
monitoring/
├── README.md
├── prometheus/
│   ├── prometheus.yml
│   └── rules/blitz.yml
├── alertmanager/
│   └── alertmanager.yml.example    # 真实配置在 .gitignore
└── grafana/
    └── dashboards/blitz.json
.github/workflows/ci.yml
snapshots/
```

---

## 本地启动

```bash
goreman -f Procfile.single start
docker compose up -d
DATABASE_URL=postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable \
ETH_HOT_WALLET_KEY=<私钥hex> \
go run cmd/wallet-service/main.go
```

---

## 监控现状

**本地 Prometheus（Homebrew）：**
- 配置路径：`/opt/homebrew/etc/prometheus/`
- rule_files：`/opt/homebrew/etc/prometheus/rules/*.yml`

**scrape jobs：**

| job | target | 状态 |
|-----|--------|------|
| prometheus | localhost:9090 | ✅ |
| etcd | localhost:2379 | ✅ |
| blitz-wallet | localhost:2112 | ✅ |
| wallet-service | localhost:2113 | ✅ |

**Alertmanager（Homebrew）：**
- 配置路径：`~/.config/alertmanager/alertmanager.yml`
- Telegram bot 已接通，测试告警验证通过

**Grafana：**
- dashboard 已导入：`monitoring/grafana/dashboards/blitz.json`
- 9个 Panel，默认时间范围 3h，30s 自动刷新

---

## etcd key 总览

| key | 用途 | TTL |
|-----|------|-----|
| `/blitz/lock/withdraw:{uid}:{chain}` | 提币分布式锁 | 30s |
| `/blitz/leader/confirm-checker` | ConfirmChecker 选主 | 15s |
| `/blitz/config/btc_rpc_host` | BTC RPC 热更新 | 永久 |
| `/blitz/config/eth_rpc_host` | ETH RPC 热更新 | 永久 |

---

## 待实现（按优先级）

1. **前端**（React + Vite + Tailwind + shadcn/ui，审美要求极高）
2. **主网切换 + 冷热钱包分离**
3. **完整 IAM 补全**（角色分配接口、审计日志、黑名单 token）
4. **user_id 统一**（TEXT → BIGINT，解锁 checkAndUpgradeLevel）
5. **ETH reorg 数据库回滚**
6. **死信队列定时重试 + 管理接口**
7. **多链扩展**（TRON/SOL/Polygon）
8. **dev-toolkit metrics 骨架**

---

## 技术决策备忘

- deposits/withdrawals.user_id 是 TEXT，users.id 是 BIGINT，暂时并存
- JWT claims 里的 user_id 是 BIGINT，提币限额从 claims.UserID 取
- NUMERIC(20,8) 存金额，sqlc 生成 string，handler 转 float64
- refresh token 旋转：每次 /refresh 撤销旧 token 换新 token
- Dead letter 重试3次（1s/2s/3s），失败写 dead_letters 表
- ETH reorg：ParentHash 校验，不匹配回退 lastHeight/lastHash
- metrics 埋点：detected/confirmed 分开计数，lock fail 在获取失败时计
- e2e/smoke 加 `//go:build e2e`，CI 只跑 integration
- alertmanager.yml 含敏感信息，加入 .gitignore，项目保留 .example

---

## 环境变量

| 变量 | 默认值 |
|------|--------|
| DATABASE_URL | postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable |
| ETH_HOT_WALLET_KEY | 空 |
| WALLET_HD_SEED | 空（测试 seed）|
| JWT_SECRET | dev-secret |
| ETCD_ENDPOINTS | localhost:2379 |
| PORT | 2113 |
| BTC_RPC_HOST | localhost:18443/wallet/blitz_wallet |
| ETH_RPC_HOST | http://localhost:8545 |

---

## 历史快照

```
snapshots/
├── SNAPSHOT-web3blitz-2026-03-18-pg-migration.md
├── SNAPSHOT-web3blitz-2026-03-18-etcd.md
├── SNAPSHOT-web3blitz-2026-03-19-auth-limits.md
├── SNAPSHOT-web3blitz-2026-03-19-rbac-swagger.md
├── SNAPSHOT-web3blitz-2026-03-19-metrics.md
└── SNAPSHOT-web3blitz-2026-03-20-monitoring.md
```
