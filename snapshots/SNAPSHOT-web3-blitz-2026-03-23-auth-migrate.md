# SNAPSHOT — web3-blitz

**里程碑**：认证全链路完善 + golang-migrate 接入
**日期**：2026-03-23

---

## 本次完成

### 认证功能
- 忘记密码全链路：`POST /api/v1/forgot-password` + `POST /api/v1/reset-password`
- QQ 邮箱 SMTP 发送重置邮件（`internal/email/email.go`，支持 465 TLS / 587 STARTTLS）
- 注册新账号：Login 页内联切换，不新开路由
- 记住我：`rememberMe` 参数控制 localStorage / sessionStorage，关标签页即清除
- 前端新增 `ForgotPassword.tsx` / `ResetPassword.tsx` 页面
- `App.tsx` 加 `/forgot-password` / `/reset-password` 公开路由

### 数据库迁移
- 引入 `golang-migrate/migrate/v4` 替代 `connect.go` 里手写 DDL
- `internal/db/migrations/` 目录，版本化管理所有 schema 变更：
  - `000001_init.up/down.sql` — 全量建表 + 种子数据
  - `000002_password_reset_tokens.up/down.sql` — 密码重置表
- 启动时自动执行未执行的迁移，幂等安全

### 工程
- `server.go` → `mux.go` 改名完成
- `connect.go` 彻底清理：去掉手写 DDL、debug log、无用查询

---

## ⚠️ 本次踩坑完整记录

### 坑 1：幽灵 postgres（第二次）
**现象**：`.env` 里 `DATABASE_URL=192.168.117.2:5432`，但容器 IP 不稳定，某次启动后该 IP 无法访问（`unexpected EOF`）。  
**根因**：Docker 容器内部 IP 在重启/网络变化后可能改变，宿主机直连容器 IP 不可靠。  
**解决**：容器已映射 `0.0.0.0:5432`，改用 `localhost:5432` 即可。幽灵 postgres 已消失，`lsof -i :5432` 确认只有 Docker 进程。  
**最终方案**：
```
DATABASE_URL=postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable&search_path=public
```
**结论**：永远用 `localhost` + 端口映射，不用容器内部 IP。

---

### 坑 2：pgx v5 search_path 问题（最坑）
**现象**：
- `docker exec psql -U blitz -d blitz -c "\dt"` 能看到 `password_reset_tokens`
- `SELECT current_database()` 返回 `blitz`
- `pg_tables` 查询也能看到表在 `public` schema
- 但 Go 程序 `SELECT COUNT(*) FROM password_reset_tokens` 报 `relation does not exist`

**排查过程**（2小时）：
1. 以为容器挂了 → 不是，`docker exec` 正常
2. 以为 sqlc 没生成 → 不是，`grep` 确认方法存在
3. 以为连错了库 → 不是，`current_database()` 返回 `blitz`
4. 以为 schema 不对 → 不是，`pg_tables` 确认在 `public`
5. 最终：在 Go 程序里直接查表，报错，而 `pg_tables` 能查到 → search_path 问题

**根因**：pgx v5 是 pgx v4 的 breaking change，连接时会执行 `SET search_path = "$user"`，重置掉 `public`。`docker exec psql` 默认 `search_path = "$user", public`，所以能找到表，而 Go 程序找不到。

**解决方案（两种都可以）**：
1. DSN 里加 `&search_path=public`
2. 在 Go 程序的同一个连接里建表（`connect.go` 里 `database.Exec("CREATE TABLE...")` — 用的是同一个连接，search_path 上下文一致）

**最终方案**：引入 golang-migrate，迁移文件通过同一个连接执行，彻底消灭这个问题。

**结论**：用 pgx v5 时，DSN 必须加 `&search_path=public`，或者用 migrate 工具管理 schema。

---

### 坑 3：`docker exec` 建表建到了哪个库？
**现象**：`docker exec psql -U blitz` 不加 `-d` 时，以为连的是 `postgres` 库，实际上 psql 会连和用户名同名的库，即 `blitz`。  
**结论**：不是建错库，是 pgx v5 search_path 问题（见坑 2）。

---

### 坑 4：golang-migrate 迁移路径
**现象**：`MIGRATIONS_PATH` 环境变量默认 `internal/db/migrations`，但 `go run` 的工作目录必须是项目根目录，否则找不到迁移文件。  
**解决**：始终在项目根目录运行 `go run ./cmd/wallet-service`，或通过 `MIGRATIONS_PATH` 显式指定绝对路径。

---

## 文件变动清单

```
新增：
- internal/email/email.go
- internal/db/migrations/000001_init.up.sql
- internal/db/migrations/000001_init.down.sql
- internal/db/migrations/000002_password_reset_tokens.up.sql
- internal/db/migrations/000002_password_reset_tokens.down.sql
- frontend/src/pages/ForgotPassword.tsx
- frontend/src/pages/ResetPassword.tsx

修改：
- internal/db/connect.go          ← 换 golang-migrate，清理所有手写 DDL
- internal/api/handler.go         ← 加 ForgotPassword / ResetPassword
- internal/api/mux.go             ← 加两条公开路由（原 server.go 改名）
- internal/db/queries.sql         ← 加 4 条密码重置查询
- internal/db/schema.sql          ← 加 password_reset_tokens 表
- frontend/src/contexts/AuthContext.tsx  ← login 加 rememberMe
- frontend/src/api/client.ts      ← token 同时查 localStorage/sessionStorage
- frontend/src/pages/Login.tsx    ← 内联注册 + 记住我 + 忘记密码跳转
- frontend/src/App.tsx            ← 加公开路由
```

---

## 本地开发启动

```bash
# .env 配置（必须在项目根目录）
DATABASE_URL=postgres://blitz:blitz@localhost:5432/blitz?sslmode=disable&search_path=public
SMTP_HOST=smtp.qq.com
SMTP_PORT=465
SMTP_USER=xxx@qq.com
SMTP_PASS=<QQ授权码>
SMTP_FROM=xxx@qq.com
FRONTEND_URL=http://localhost:5173

# 启动（在项目根目录）
docker compose up -d postgres
go run ./cmd/wallet-service   # 自动执行迁移
cd frontend && npm run dev
```

---

## 历史快照

```
snapshots/
└── SNAPSHOT-web3-blitz-2026-03-23-auth-migrate.md   ← 本次
```
