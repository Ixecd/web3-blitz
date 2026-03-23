归档说明：此目录存放历史快照，文件命名格式为 SNAPSHOT-{项目}-{日期}-{里程碑}.md

# SNAPSHOT — web3-blitz

**里程碑**：登录认证全链路打通 + 管理后台加载成功
**日期**：2026-03-23

---

## 本次完成

### 后端
- `users` 表加了 `email` 字段（NOT NULL UNIQUE）
- `Register` / `Login` 接口从 `username` 改成 `email`
- `sqlc` 新增 `GetUserByEmail` 查询，`CreateUser` 加 `email` 参数
- `connect.go` 清理了重复混乱的 migration 代码，改成干净的三行
- `connect.go` DSN 改成 `192.168.117.2:5432`（见下方重要说明）
- `server.go` 改名为 `mux.go`（待执行）
- roles / permissions / role_permissions 初始化 SQL 加入 `schema.sql`

### 前端
- `AuthContext.tsx`：`username` 全部改成 `email`
- `client.ts`：修复 `json.code !== undefined && json.code !== 0` 的判断
- `Layout.tsx`：去掉「已登录」文字，Logo 放大，BLITZ 文字颜色修复
- `Admin.tsx` 表头颜色从 `text-faint` 改为 `text-muted`
- `Login.tsx`：邮件地址登录、忘记密码、记住我、眼睛切换密码、错误提示不抖动

---

## ⚠️ 重要：幽灵 postgres 问题

**现象**：Go 连 `localhost:5432` 时连的是 macOS 本地某个 postgres（现已消失），
不是 Docker 容器，导致数据完全不一致，折腾了两个小时。

**当前解决方案**：`internal/db/connect.go` 默认 DSN 写死了容器 IP：
```go
dsn = "postgres://blitz:blitz@192.168.117.2:5432/blitz?sslmode=disable"
```

**正确做法（待完成）**：
1. `connect.go` 恢复成 `localhost`
2. 项目根目录加 `.env` 文件（已加入 `.gitignore`）：
```
DATABASE_URL=postgres://blitz:blitz@192.168.117.2:5432/blitz?sslmode=disable
```
3. `main.go` 用 `godotenv` 加载 `.env`

容器 IP 可能变，用 `docker inspect postgres | grep IPAddress` 确认。

---

## 本地开发启动步骤

```bash
# 1. 起基础设施
docker compose up -d postgres
goreman -f Procfile.single start   # 起 etcd（另一个终端）

# 2. 起后端（确保 DATABASE_URL 指向正确的容器）
export DATABASE_URL="postgres://blitz:blitz@192.168.117.2:5432/blitz?sslmode=disable"
go run ./cmd/wallet-service

# 3. 起前端（另一个终端）
cd frontend && npm run dev
```

---

## 数据库初始化（首次或重建 volume 后）

```bash
# 执行 schema（包含角色权限初始化）
docker exec -i postgres psql -U blitz -d blitz -f /docker-entrypoint-initdb.d/schema.sql

# 注册第一个管理员
curl -X POST http://localhost:2113/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"email":"superadmin@blitz.com","password":"admin123456"}'

# 分配 admin 角色（替换 user_id 为实际返回的 id）
docker exec -i postgres psql -U blitz -d blitz -c "
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r
WHERE u.email='superadmin@blitz.com' AND r.name='admin'
ON CONFLICT DO NOTHING;"
```

---

## 待完成

### 高优先级
- [ ] `connect.go` DSN 改回 localhost + `.env` 方案
- [ ] 清理 handler.go 里的临时 debug log（user记录、注册前等）
- [ ] Login.tsx 密码 autofill 显示问题（浏览器安全限制，低优先级）

### 页面
- [ ] Dashboard 页面按新设计语言重做
- [ ] Deposit（充值）页面重做
- [ ] Withdraw（提币）页面重做
- [ ] Admin 页面用户列表显示 email 而非 username

### 功能
- [ ] 记住我 cookie 逻辑
- [ ] 忘记密码页面
- [ ] 注册新账号页面
- [ ] 扫码登录

### 工程
- [ ] RBAC 角色初始化移入 `runSeed`，不依赖手动执行
- [ ] etcd 里的旧 registry 数据清理
- [ ] server.go 改名为 mux.go

---

## 文件变动清单

```
内部修改（需手动覆盖）：
- internal/db/connect.go       ← DSN + migration 清理
- internal/db/schema.sql       ← 加了 email 字段 + 角色权限初始化
- internal/db/queries.sql      ← GetUserByEmail + CreateUser 加 email
- internal/api/handler.go      ← Register/Login 改 email，含临时 debug log
- frontend/src/contexts/AuthContext.tsx   ← username → email
- frontend/src/api/client.ts   ← 修复错误判断
- frontend/src/components/Layout.tsx     ← 去掉已登录，Logo 修复
- frontend/src/index.css       ← data-table th 颜色提亮
- frontend/src/pages/Login.tsx ← 邮件登录 + 忘记密码 + 记住我
```
