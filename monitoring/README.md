# monitoring

本地开发监控栈配置文件（Homebrew 安装）。

## 目录结构

```
monitoring/
├── prometheus/
│   ├── prometheus.yml        # scrape + alerting 配置
│   └── rules/
│       └── blitz.yml         # 6 条业务告警规则
├── alertmanager/
│   └── alertmanager.yml      # Telegram 告警路由（⚠️ bot_token 需本地填写，不提交）
└── grafana/
    └── dashboards/           # dashboard JSON（待导出）
```

## 本地部署（Homebrew）

```bash
# 1. 拷贝 prometheus 配置
cp monitoring/prometheus/prometheus.yml /opt/homebrew/etc/prometheus/prometheus.yml
cp monitoring/prometheus/rules/blitz.yml /opt/homebrew/etc/prometheus/rules/blitz.yml

# 2. 将 rule_files 改为绝对路径（prometheus.yml 第8行）
#    rule_files:
#      - /opt/homebrew/etc/prometheus/rules/*.yml

# 3. 重启 Prometheus
brew services restart prometheus

# 4. 拷贝 alertmanager 配置（填写 bot_token 后）
cp monitoring/alertmanager/alertmanager.yml /opt/homebrew/etc/alertmanager/alertmanager.yml
brew services restart alertmanager
```

## 告警规则一览

| 规则 | 级别 | 触发条件 |
|------|------|---------|
| WithdrawBroadcastFailed | critical | 5分钟内提币失败 > 0，持续1分钟 |
| DeadLetterQueued | warning | 5分钟内死信队列新增 > 0，持续1分钟 |
| ETHReorgDetected | warning | 10分钟内检测到reorg，立即触发 |
| LockAcquireFailSpike | warning | 5分钟内锁获取失败 > 10次，持续2分钟 |
| WalletServiceDown | critical | wallet-service 不可达，持续1分钟 |
| EtcdDown | critical | etcd 不可达，持续1分钟 |

## ⚠️ 注意

- `alertmanager.yml` 含 Telegram `bot_token` 和 `chat_id`，已加入 `.gitignore`
- 项目中保留 `alertmanager.yml.example` 作为模板
