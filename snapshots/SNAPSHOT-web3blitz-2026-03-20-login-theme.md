归档说明：此目录存放历史快照，文件命名格式为 SNAPSHOT-{项目}-{日期}-{里程碑}.md

# SNAPSHOT — web3-blitz 前端

**里程碑**：登录页 UI 打磨 + 主题系统落地
**日期**：2026-03-20

## 已完成

### 主题系统
- 5套主题：Void / Glacier / Grove / Ember / Aurum，CSS 变量驱动
- ThemeContext + ThemeSwitcher，localStorage 持久化
- --color-grid 独立变量，暗色白线/亮色 accent

### 登录页
- 左右分屏，动态光晕背景（3 orb，4s/5s/6s）+ 网格呼吸
- 「欢迎回来。」绝对定位左下角 clamp 响应式
- 邮件 + 密码眼睛切换 + 忘记密码 + 记住我
- 错误固定 16px 高不顶布局，按钮 submit 时 ref 读值
- or 分割线 + 扫码 + 注册，响应式 768px 折叠

### 工程
- vite-env.d.ts，tsconfig @/* 别名，CI branch → Master

## 待完成
- 密码 autofill 重新实现
- 记住我 / 忘记密码 / 注册 / 扫码逻辑
- 其余页面按新设计语言重做
- Layout 适配主题，全页面跨主题验收