# web3-blitz ⚡

> 基于 Go + K8s 的 Web3 充提币系统，支持 BTC/ETH 充值监控、HD 钱包派生、提币审核、RBAC 权限管理。

## 🏗️ 架构

```
┌─────────────────────────────────────────────────────┐
│                    K8s Cluster                      │
│  namespace: web3-blitz                              │
│                                                     │
│  ┌────────────┐  ┌──────────────┐  ┌─────────────┐  │
│  │  bitcoind  │  │   geth-rpc   │  │   postgres  │  │
│  │ (regtest)  │  │  (dev mode)  │  │  (PVC 1Gi)  │  │
│  └─────┬──────┘  └──────┬───────┘  └──────┬──────┘  │
│        │                │                 │         │
│  ┌─────▼────────────────▼─────────────────▼──────┐  │
│  │                wallet-service                 │  │
│  │  HD派地址 │ Deposit Watcher │ ConfirmChecker   │  │
│  │  JWT 认证  │  RBAC  │  忘记密码  │  REST API    │  │
│  └───────────────────────────────────────────────┘  │
│                         │                           │
│                  ┌──────▼──────┐                    │
│                  │    etcd     │                    │
│                  │  (选主/锁)   │                    │
│                  └─────────────┘                    │
└─────────────────────────────────────────────────────┘
```

## ✨ 功能

- **HD 钱包**：BIP44 路径派生，支持 BTC/ETH 充值地址生成
- **Deposit Watcher**：实时扫块监听充值，自动入账，支持确认数检查
- **ConfirmChecker**：etcd 选主，分布式环境下只有一个节点执行确认逻辑
- **提币**：余额校验、日限额、分布式锁防重复提交
- **死信队列**：deposit 写入失败兜底，保证数据不丢
- **JWT 认证**：access token + refresh token 轮转，记住我
- **RBAC 权限**：roles / permissions / user_roles，支持管理员/运营/普通用户
- **忘记密码**：QQ 邮箱 SMTP，token 30 分钟有效，防枚举攻击
- **数据库迁移**：golang-migrate + embed.FS，迁移文件打进二进制，启动自动执行
- **云原生**：一键 `dtk deploy` 部署到 K8s，零外部 Helm 依赖

## 🚀 快速开始

### 本地开发

```bash
# 启动依赖
docker compose up -d

# 配置环境变量（复制后按需修改）
cp .env.example .env

# 启动服务（自动执行数据库迁移）
go run ./cmd/wallet-service
```

### 一键部署到 K8s

```bash
# 确保 configs/project.env 配置正确
dtk deploy
```

### 常用接口

```bash
# 注册
curl -X POST http://localhost:2113/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"username":"test","email":"test@example.com","password":"123456"}'

# 登录
curl -X POST http://localhost:2113/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"123456"}'

# 生成充值地址
curl -X POST http://localhost:2113/api/v1/address \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"chain":"btc"}'
```

## 📦 服务说明

| 服务 | 端口 | 说明 |
|------|------|------|
| wallet-service | 2113 | 核心钱包服务，REST API |
| bitcoind | 18443 | BTC 节点 (regtest) |
| geth-rpc | 8545 | ETH 节点 (dev mode) |
| postgres | 5432 | 业务数据库 (PVC 持久化) |
| etcd | 2379 | 分布式协调（选主、分布式锁） |

## 🗂️ 项目结构

```
web3-blitz/
├── cmd/
│   └── wallet-service/         # 服务入口
├── internal/
│   ├── api/                    # HTTP handler + mux
│   ├── db/                     # sqlc 生成的 DB 层
│   │   └── migrations/         # golang-migrate 迁移文件
│   ├── email/                  # SMTP 邮件发送
│   └── wallet/                 # HD 钱包、Watcher、BTC/ETH 实现
├── deployments/
│   └── web3-blitz/             # Helm Chart（自包含，零外部依赖）
│       └── templates/
│           ├── postgres-statefulset.yaml
│           ├── etcd-deployment.yaml
│           ├── wallet-service-deployment.yaml
│           ├── bitcoind-deployment.yaml
│           └── geth-deployment.yaml
├── build/docker/wallet-service/ # Dockerfile
├── configs/                    # project.env、components.yaml
├── snapshots/                  # 里程碑快照归档
└── docs/                       # 设计文档
```

## ⚙️ 环境变量

`.env`（本地开发）：

```ini
DATABASE_URL=postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable&search_path=public
ETCD_ENDPOINTS=localhost:2379
JWT_SECRET=your-secret-key
WALLET_HD_SEED=your-hd-seed

# 邮件（忘记密码功能）
SMTP_HOST=smtp.qq.com
SMTP_PORT=465
SMTP_USER=xxx@qq.com
SMTP_PASS=your-auth-code
SMTP_FROM=xxx@qq.com
FRONTEND_URL=http://localhost:5173
```

`configs/project.env`（K8s 部署）：

```ini
PROJECT_NAME=web3-blitz
KUBE_NAMESPACE=web3-blitz
REGISTRY_PREFIX=qingchun22
ARCH=arm64
VERSION=v0.1.4
```

## 🔄 数据库迁移

迁移文件在 `internal/db/migrations/`，服务启动时自动执行：

```
000001_init.up.sql              # 全量建表 + 种子数据
000002_password_reset_tokens.up.sql
```

新增表只需创建迁移文件，重启服务自动执行，无需手动操作。

## 📚 文档

- [Helm Chart 设计](docs/design/helm-chart-design.md)
- [快照归档](snapshots/)
- [TODO](TODO.md)

## 📄 License

MIT
