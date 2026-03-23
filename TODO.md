# TODO — web3-blitz 路线图

> 充提币系统，从开发环境走向可交付。
> 按优先级排列，持续更新。

---

## 🔴 高优先级

### 工程
- [ ] RBAC 角色初始化移入迁移文件（`000001_init.up.sql` 已包含，`runSeed` 可以删了）
- [ ] etcd 旧 registry 数据清理
- [ ] `connect.go` 里遗留的 `runSeed` 函数删除（seed 已在 migration 里）
- [ ] 把 debug 用的临时代码从 `connect.go` 里清理（`current_database` 查询等）

### 功能
- [ ] 记住我 — cookie 方案（当前用 localStorage/sessionStorage，考虑 httpOnly cookie 更安全）
- [ ] 忘记密码 — 限流（同一邮箱 1 分钟内只能发一次）
- [ ] 注册 — 邮箱验证码验证

---

## 🟡 中优先级

### 页面
- [ ] Dashboard 页面按新设计语言重做
- [ ] Deposit（充值）页面重做
- [ ] Withdraw（提币）页面重做
- [ ] Admin 用户列表显示 email 而非 username

### 功能
- [ ] 扫码登录
- [ ] refresh token 自动续期（前端拦截 401 自动刷新）
- [ ] 登出时踢掉所有设备选项

### 工程
- [ ] `handler.go` 拆分：auth、wallet、admin 各自独立文件
- [ ] 统一错误码文档（`internal/pkg/code/` 补全所有错误码说明）

---

## 🟢 低优先级

- [ ] 提币审核流程（pending → 人工审核 → 广播）
- [ ] 充值地址二维码生成
- [ ] 交易记录导出 CSV
- [ ] 管理后台操作日志
- [ ] 多币种扩展（USDT、TRX 等）

---

## ✅ 已完成

- [x] HD 钱包生成（BTC + ETH）
- [x] 充值地址生成 + 注册表恢复
- [x] BTC / ETH 充值扫块监听
- [x] 确认数检查（ConfirmChecker，etcd 选主）
- [x] 提币接口（分布式锁防重、余额校验、日限额）
- [x] 死信队列（deposit 写入失败兜底）
- [x] JWT 认证 + refresh token 轮转
- [x] RBAC 权限系统（roles / permissions / user_roles）
- [x] 管理后台基础（用户列表、等级升级、限额配置）
- [x] slog 结构化日志（JSON，LOG_LEVEL=debug）
- [x] `register` / `login` 改用 email
- [x] 忘记密码全链路（QQ 邮箱 SMTP）
- [x] 注册新账号（Login 页内联切换）
- [x] 记住我（localStorage / sessionStorage 切换）
- [x] golang-migrate 数据库迁移（替代手写 DDL）
- [x] server.go → mux.go 改名

---

> 每完成一项，移到 ✅ 已完成，并更新 SNAPSHOT。
