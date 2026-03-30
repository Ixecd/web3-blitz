# SNAPSHOT — web3-blitz

> 项目整体快照，新会话开始时直接扔给 Claude，5 秒对齐，继续工作。
> 最后更新：2026-03-30 / v0.1.11

---

## 项目是什么

BTC/ETH 充提币系统，同时作为 dtk 的活体验证环境。

**仓库**：github.com/Ixecd/web3-blitz
**部署**：`dtk deploy`，namespace: web3-blitz

---

## 当前 K8s 状态

```
namespace: web3-blitz
  dev-toolkit-controller（Deployment）  1/1 Running ✅  自愈控制器
  wallet-service（Deployment ×2）       1/1 Running ✅  核心业务
  web3-blitz-etcd-0（StatefulSet）      1/1 Running ✅  PVC 持久化
  web3-blitz-postgres-0（StatefulSet）  1/1 Running ✅  PVC 持久化
```

---

## 服务拓扑（components.yaml）

```yaml
components:
  - name: web3-blitz-postgres   # 层级 1（并行）
    type: statefulset
  - name: web3-blitz-etcd       # 层级 1（并行）
    type: statefulset
  - name: wallet-service        # 层级 2
    type: deployment
    depends_on: [web3-blitz-postgres, web3-blitz-etcd]
  - name: web3-blitz-controller # 层级 3
    type: deployment
    depends_on: [web3-blitz-etcd]
```

---

## 关键配置

### project.env

```ini
PROJECT_NAME=web3-blitz
KUBE_NAMESPACE=web3-blitz
REGISTRY_PREFIX=qingchun22
ARCH=arm64
VERSION=v0.1.11
ETCD_ENDPOINTS=web3-blitz-etcd:2379
```

### K8s Secret（手动创建，不进 git）

```bash
./scripts/create-secret.sh   # 从 .env 读取，幂等
```

包含：DATABASE_URL / WALLET_HD_SEED / SMTP_PASS / JWT_SECRET

### values.yaml env（非敏感，进 git）

ETCD_ENDPOINTS / BTC_NETWORK / SMTP_HOST / SMTP_PORT / SMTP_USER / SMTP_FROM / FRONTEND_URL

---

## 代码结构

```
web3-blitz/
├── cmd/wallet-service/         # 服务入口
├── internal/
│   ├── api/
│   │   ├── handler.go          # Handler struct + NewHandler
│   │   ├── auth.go             # Register/Login/Refresh/Logout/GetMe/ForgotPassword/ResetPassword
│   │   ├── wallet.go           # GenerateAddress/GetBalance/ListDeposits/Withdraw/ListWithdrawals
│   │   ├── admin.go            # ListUsers/UpgradeUser/ListWithdrawalLimits/UpdateWithdrawalLimit
│   │   └── mux.go              # 路由注册
│   ├── db/migrations/          # golang-migrate，启动自动执行
│   ├── email/                  # SMTP 邮件
│   ├── lock/                   # 分布式锁
│   └── wallet/btc/ eth/        # HD 钱包、Watcher
├── deployments/web3-blitz/
│   ├── web3-blitz-postgres/    # StatefulSet
│   ├── web3-blitz-etcd/        # StatefulSet（PVC 持久化）
│   ├── wallet-service/         # Deployment，secretKeyRef 注入敏感变量
│   └── web3-blitz-controller/  # A2 自愈控制器
├── configs/
│   ├── project.env
│   ├── components.yaml
│   └── resources.yaml          # controller 监控配置
└── scripts/
    └── create-secret.sh        # 幂等创建 K8s Secret
```

---

## API 概览

```
POST /api/v1/register           # 注册
POST /api/v1/login              # 登录
POST /api/v1/refresh            # 刷新 token
POST /api/v1/logout             # 登出
GET  /api/v1/me                 # 当前用户信息
POST /api/v1/forgot-password    # 忘记密码
POST /api/v1/reset-password     # 重置密码

POST /api/v1/address            # 生成充值地址（BTC/ETH）
GET  /api/v1/balance            # 查询余额
GET  /api/v1/deposits           # 充值历史
POST /api/v1/withdraw           # 发起提币
GET  /api/v1/withdrawals        # 提币历史

GET  /api/v1/admin/users        # 用户列表（Admin）
POST /api/v1/admin/users/:id/upgrade   # 升级用户等级
GET  /api/v1/admin/limits       # 提币限额配置
PUT  /api/v1/admin/limits       # 更新限额
```

---

## 接下来（P0）

1. 充值流程联调（GenerateAddress → Deposit Watcher → 入账确认）
2. 提币流程联调（余额校验 → 广播 → 状态追踪）
3. Dashboard 接真实数据

---

## 历史快照

```
snapshots/
├── SNAPSHOT-web3-blitz-2026-03-24-fully-deployed.md
├── SNAPSHOT-web3-blitz-2026-03-25-controller.md
└── SNAPSHOT-web3-blitz-2026-03-30-k8s-hardening.md
```
