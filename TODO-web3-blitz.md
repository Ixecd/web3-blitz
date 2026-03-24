# TODO — web3-blitz 路线图

> 充提币系统，从开发环境走向可交付。
> 按优先级排列，持续更新。

---

## 🔴 P0 — 核心功能联调

### 充值流程
- [ ] Deposit 页面接真实 API
- [ ] 生成 BTC / ETH 充值地址展示
- [ ] 充值历史列表
- [ ] 到账确认逻辑验证（regtest 环境）

### 提币流程
- [ ] Withdraw 页面接真实 API
- [ ] 余额校验、限额校验联调
- [ ] 提币状态追踪（pending → completed / failed）

### Dashboard
- [ ] 接真实数据：余额、最近充值、最近提币
- [ ] 数据定时刷新

### 工程修复
- [ ] etcd 旧 registry 数据清理
- [ ] `handler.go` 拆分：auth、wallet、admin 各自独立文件
- [ ] wallet-service 环境变量改用 K8s Secret（DATABASE_URL、JWT_SECRET、SMTP_PASS）

---

## 🟡 P1 — 完善

### 账户体系
- [ ] 忘记密码限流（同一邮箱 1 分钟内只能发一次）
- [ ] 注册邮箱验证码验证
- [ ] refresh token 自动续期（前端拦截 401 自动刷新）
- [ ] 登出时踢掉所有设备选项

### Admin
- [ ] 用户列表显示 email 而非 username
- [ ] 升级用户等级联调验证

### K8s
- [ ] SMTP 配置注入（SMTP_HOST / SMTP_USER / SMTP_PASS / SMTP_FROM）
- [ ] etcd 部署到 K8s（当前 emptyDir，重启数据丢失）

---

## 🟢 P2 — 收尾

- [ ] 扫码登录
- [ ] 所有页面跨主题验收
- [ ] Layout 侧边栏适配所有主题
- [ ] 提币审核流程（pending → 人工审核 → 广播）
- [ ] 充值地址二维码生成
- [ ] 交易记录导出 CSV
- [ ] 管理后台操作日志
- [ ] 多币种扩展（USDT、TRX 等）
- [ ] 用 dtk 部署 web3-blitz 到本地 K8s，跑通完整流程，作为 dtk 的 demo

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
- [x] register / login 改用 email
- [x] 忘记密码全链路（QQ 邮箱 SMTP）
- [x] 注册新账号（Login 页内联切换）
- [x] 记住我（localStorage / sessionStorage 切换）
- [x] golang-migrate + embed.FS（迁移文件打进二进制，启动自动执行）
- [x] server.go → mux.go 改名
- [x] 自包含 Helm chart（postgres StatefulSet + etcd Deployment + 业务服务，零外部依赖）
- [x] initContainers 启动顺序（wait-postgres + wait-etcd）
- [x] K8s 全链路部署跑通（5 个 pod 全部 Running，零重启）
- [x] /healthz 路由（dtk 状态机 VALIDATING 验证需要）
- [x] wallet-service-deployment.yaml probe 改为 /healthz
- [x] CI 修复（golang-migrate 替换 psql schema.sql，search_path=public）
- [x] .gitignore 加 data/（etcd 本地数据目录）
- [x] pgx v5 search_path 问题修复（DSN 加 &search_path=public）

---

> 两个项目的交汇点：web3-blitz 跑通之后就是 dtk 最好的推广 demo，部署过程中发现的问题直接反哺 DTK P0/P1。

> 每完成一项，移到 ✅ 已完成，并更新 SNAPSHOT。
