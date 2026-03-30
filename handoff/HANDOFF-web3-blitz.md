# 项目交接文档 — web3-blitz

> 写给下一个 Claude
> 日期：2026-03-30
> 版本：v0.1.11

---

## 写在前面

web3-blitz 是 BTC/ETH 充提币系统，同时是 dtk 的活体验证环境。两个项目同步开发，web3-blitz 的部署问题直接反哺 dtk。

**qc 的工作风格（认真读）**：
- 设计优先，代码其次。不要上来就写代码，先对齐设计再动手
- 每完成一个里程碑：commit → SNAPSHOT → 更新 TODO
- 喜欢被推 back，不喜欢被一味认同。他通常是对的
- `slog` 不用 `log`，kubectl CLI 不用 client-go，严格分包
- 前端暂时不是重点，先把后端链路跑稳

---

## 一、当前状态

**K8s（全部 Running）**：

```
namespace: web3-blitz
  dev-toolkit-controller    1/1 Running  ✅
  wallet-service (×2)       1/1 Running  ✅
  web3-blitz-etcd-0         1/1 Running  ✅  StatefulSet + PVC
  web3-blitz-postgres-0     1/1 Running  ✅  StatefulSet + PVC
```

**已验证**：
- dtk deploy 全链路 ✅
- controller 自愈（wallet-service 删除后 ~13s 恢复）✅
- 忘记密码 SMTP 在 K8s 跑通 ✅
- K8s Secret 注入（DATABASE_URL / WALLET_HD_SEED / SMTP_PASS / JWT_SECRET）✅

---

## 二、目录结构

```
web3-blitz/
├── cmd/wallet-service/main.go
├── internal/
│   ├── api/
│   │   ├── handler.go      # Handler struct + NewHandler
│   │   ├── auth.go         # 认证相关 handler
│   │   ├── wallet.go       # 充提币 handler
│   │   ├── admin.go        # 管理后台 handler
│   │   └── mux.go          # 路由注册
│   ├── auth/               # JWT + RBAC 中间件
│   ├── db/
│   │   ├── migrations/     # golang-migrate，启动自动执行
│   │   └── queries/        # sqlc 生成
│   ├── email/              # SMTP 邮件（QQ 邮箱）
│   ├── lock/               # etcd 分布式锁
│   └── wallet/
│       ├── btc/            # BTC HD 钱包 + Watcher
│       ├── eth/            # ETH HD 钱包 + Watcher
│       └── types/          # 公共类型
├── deployments/web3-blitz/
│   ├── web3-blitz-postgres/
│   ├── web3-blitz-etcd/    # StatefulSet，PVC 1Gi
│   ├── wallet-service/     # secretKeyRef 注入敏感变量
│   └── web3-blitz-controller/
├── configs/
│   ├── project.env
│   ├── components.yaml
│   └── resources.yaml
└── scripts/
    └── create-secret.sh    # 幂等创建 K8s Secret
```

---

## 三、部署流程

```bash
# 1. 创建/更新 Secret（只需要 .env 在本地）
./scripts/create-secret.sh

# 2. 部署
dtk deploy
```

Secret 手动管理，不进 git，不被 helm 管理。

---

## 四、接下来（P0）

**充值联调**：
1. `GenerateAddress` 接口确认能生成正确地址
2. regtest 环境发送 BTC/ETH，验证 Deposit Watcher 能检测到
3. ConfirmChecker 确认数达到后入账

**提币联调**：
1. `Withdraw` 接口余额校验、限额校验
2. 广播到 regtest 节点
3. 状态追踪（pending → completed / failed）

**后续**：
- Dashboard 接真实数据
- Prometheus + Grafana 监控
- 主网联调

---

## 五、常用命令

```bash
# 部署
cd ~/web3-blitz && dtk deploy

# 查看状态
dtk status
kubectl get pods -n web3-blitz

# 查看日志
kubectl logs -n web3-blitz deployment/wallet-service -f

# 本地调试
kubectl port-forward -n web3-blitz deployment/wallet-service 2113:2113

# 重建 secret（密码轮转时用）
./scripts/create-secret.sh
```

---

## 六、已知问题

| # | 问题 | 优先级 |
|---|------|--------|
| 1 | etcd 旧 registry 数据需要清理 | P1 |
| 2 | ETH geth 403 Forbidden（geth 配置问题） | P1 |
| 3 | FRONTEND_URL 写死 localhost:5173，上主网前改 | P1 |
| 4 | controller RBAC 权限过宽，生产环境需要收紧 | P2 |
