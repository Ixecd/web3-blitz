# SNAPSHOT — web3-blitz

**里程碑**：K8s 工程化完成（Secret / etcd PVC / handler 拆分）
**日期**：2026-03-30
**版本**：v0.1.11

---

## 本次完成

### handler.go 拆分

829 行 `internal/api/handler.go` 按职责拆成四个文件，同一 package，直接建文件剪函数：

```
internal/api/
├── handler.go    # Handler struct + NewHandler
├── auth.go       # Register, Login, Refresh, Logout, GetMe, ForgotPassword, ResetPassword
├── wallet.go     # GenerateAddress, GetBalance, ListDeposits, GetTotalBalance, Withdraw, ListWithdrawals
└── admin.go      # ListUsers, UpgradeUser, ListWithdrawalLimits, UpdateWithdrawalLimit
```

### etcd StatefulSet + PVC

从 Deployment + emptyDir 迁移到 StatefulSet + PVC，重启数据不再丢失：

```yaml
volumeClaimTemplates:
  - metadata:
      name: etcd-data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 1Gi
```

验证：`web3-blitz-etcd-0` 挂载 PVC `etcd-data-web3-blitz-etcd-0` ✅

### K8s Secret

敏感变量从 values.yaml 明文迁移到 K8s Secret：

| 变量 | 存储位置 |
|------|---------|
| DATABASE_URL | K8s Secret |
| WALLET_HD_SEED | K8s Secret |
| SMTP_PASS | K8s Secret |
| JWT_SECRET | K8s Secret |
| ETCD_ENDPOINTS / BTC_NETWORK / SMTP_HOST 等 | values.yaml env 块 |

`deployment.yaml` 混合写法：

```yaml
env:
  - name: DATABASE_URL
    valueFrom:
      secretKeyRef:
        name: wallet-service-secret
        key: DATABASE_URL
  # ... 其他 secret
  {{- with .Values.env }}
  {{- toYaml . | nindent 12 }}
  {{- end }}
```

`scripts/create-secret.sh` 幂等创建/更新（从 .env 读，CI/CD 从环境变量读）。

### SMTP 在 K8s 验证通过

忘记密码全链路在 K8s 环境验证：
- 注册账号 → 请求重置 → 收到邮件 ✅
- 邮件含正确 token，30 分钟有效

---

## 文件变动

```
新增：
- internal/api/auth.go
- internal/api/wallet.go
- internal/api/admin.go
- scripts/create-secret.sh

修改：
- internal/api/handler.go          ← 只保留 struct + NewHandler
- deployments/web3-blitz/web3-blitz-etcd/templates/deployment.yaml → statefulset.yaml
- deployments/web3-blitz/wallet-service/templates/deployment.yaml  ← secretKeyRef
- deployments/web3-blitz/wallet-service/values.yaml                ← 删除敏感 env
- configs/components.yaml          ← etcd type 改为 statefulset
```

---

## 当前 K8s 状态

```
namespace: web3-blitz
  dev-toolkit-controller-xxx    1/1 Running  ✅
  wallet-service-xxx (×2)       1/1 Running  ✅
  web3-blitz-etcd-0             1/1 Running  ✅  StatefulSet + PVC
  web3-blitz-postgres-0         1/1 Running  ✅  StatefulSet + PVC
```

---

## 历史快照

```
snapshots/
├── SNAPSHOT-web3-blitz-2026-03-24-fully-deployed.md
├── SNAPSHOT-web3-blitz-2026-03-25-controller.md
└── SNAPSHOT-web3-blitz-2026-03-30-k8s-hardening.md  ← 本次
```
