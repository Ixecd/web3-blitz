# GOTCHAS — 部署踩坑记录

## 2026-05-16 e2e 首次部署

使用 `kp deploy` 将 Blitz 部署到 macOS orbstack K8s 集群时遇到的完整问题链。

### 1. Dockerfile 构建目标不存在

```
stat /app/cmd/blitz: directory not found
```

**根因**: `build/docker/blitz/Dockerfile` 构建目标是 `./cmd/blitz`，但项目 cmd 目录下只有 `wallet-service`、`chain-miner`、`encrypt-seed`、`pos-sim`，没有 `cmd/blitz`。

**修复**: 改为 `./cmd/wallet-service`。

### 2. CVE 阻断部署

```
[HIGH] CVE-2026-29181 go.opentelemetry.io/otel
```

**根因**: `go.opentelemetry.io/otel` v1.40.0 存在远程 DoS 漏洞（v1.36.0–v1.40.0 受影响）。

**修复**: `go get go.opentelemetry.io/otel@v1.41.0`。

**附加坑**: CD tool 的 `trivy image` 扫描发生在 `docker build` 之前。如果远程 registry 已有旧镜像（同名 tag），trivy 会拉到旧镜像扫出 CVE → 阻断 → 永远不进 build → 死锁。解决：手动 `docker build && docker push` 覆盖旧镜像，或 bump VERSION。

### 3. `deps:` YAML 字段名不匹配

```
拓扑排序只产 1 层，blitz-postgres / blitz-etcd / blitz 并行部署
```

**根因**: `configs/components.yaml` 用 `deps:`，但 Go 结构体标签是 `yaml:"depends_on"`。依赖关系从未被解析，所有服务入度为 0，全部堆在同一层。

**修复**: 改为 `depends_on:`。修复后自动拆成 2 层（Layer 0: postgres + etcd，Layer 1: blitz）。

### 4. Deployment 缺少关键环境变量

Pod CrashLoopBackOff，日志:
```
读取种子解密密码失败: inappropriate ioctl for device
```

**根因**:
- `SEED_PASSPHRASE` — 代码先读环境变量，没有则交互式输入。容器无 TTY 导致崩溃。
- `JWT_SECRET` — 硬要求，未设置则 os.Exit(1)。
- `DATABASE_URL` — 代码读此变量，但 deployment 只设了 `DB_USER/PASSWORD/HOST/NAME/SSLMODE` 独立变量。

**修复**: deployment template 补上了这三个 env，从 `blitz-secret` 引用。

**注意**: CD tool 的 `ensureSecret` 和 `scripts/create-secret.sh` 自动创建的 `blitz-secret` **不包含** `SEED_PASSPHRASE`，需要手动添加。

### 5. HD 种子文件未挂载

```
加密种子文件 configs/secrets/hd-seed.enc 不存在
```

**根因**: HD 种子文件（`configs/secrets/hd-seed.enc`）需要先由 `encrypt-seed` 工具生成，然后挂载到容器。

**修复**:
1. `go run ./cmd/encrypt-seed/gen.go` 生成加密种子
2. `kubectl create secret generic blitz-hd-seed --from-file=hd-seed.enc=...`
3. deployment 添加 volume + volumeMount

### 6. Secret Volume 权限不匹配

```
open configs/secrets/hd-seed.enc: permission denied
```

**根因**: deployment 的 `securityContext.runAsUser: 1000`，但 secret volume `defaultMode: 0400` 只允许 root 读。

**修复**: `defaultMode` 改为 `0444`。

### 7. Volume 挂载路径错误

**根因**: Dockerfile 使用的是 `FROM scratch`，WORKDIR 默认是 `/`。代码中相对路径 `configs/secrets/hd-seed.enc` 解析到 `/configs/secrets/`，但 volume mountPath 写的是 `/app/configs/secrets`。

**修复**: mountPath 改为 `/configs/secrets`。

### 8. 数据库迁移文件未打包

```
migrate init: failed to open source, "file://internal/db/migrations": no such file or directory
```

**根因**: Dockerfile 只把编译好的二进制 `blitz` COPY 进 scratch 镜像，`internal/db/migrations/*.sql` 没包含。

**修复**: Dockerfile 添加 `COPY --from=builder /app/internal/db/migrations /migrations`，deployment 加 `MIGRATIONS_PATH=/migrations` 环境变量。

### 9. DATABASE_URL 中 $(DB_PASSWORD) 不展开

**根因**: Kubernetes 的 `$(VAR)` 语法在 env `value` 字段中不可用于跨 `valueFrom` 引用。

**修复**: 改用 `valueFrom.secretKeyRef` 直接从 `blitz-secret` 取。

---

## 检查清单（新服务接入 kp deploy 前）

- [ ] `build/docker/<name>/Dockerfile` 的构建目标是否存在
- [ ] `configs/components.yaml` 用 `depends_on`（不是 `deps`）声明依赖
- [ ] 基础设施服务（DB、etcd 等）配置在 Layer 0
- [ ] deployment template 覆盖所有必需的 env（检查 `main.go` 中 `os.Getenv` + `os.Exit`）
- [ ] scratch 镜像：二进制文件、迁移 SQL、CA 证书、种子文件全部 COPY
- [ ] `runAsUser` 与 volume `defaultMode` 不冲突
- [ ] 相对路径的 mountPath 匹配 WORKDIR
- [ ] `scripts/create-secret.sh` 包含所有必需 secret key
- [ ] `kp deploy` 前先 `go mod tidy` + CVE 扫描
