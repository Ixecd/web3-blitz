# etcd 使用指南

> web3-blitz 中 etcd 的本地启动、验证和操作手册。

---

## 本地启动

### 单节点（开发推荐）

```bash
goreman -f Procfile.single start
```

`Procfile.single` 内容：
```
etcd1: etcd --name infra1 \
  --listen-client-urls http://127.0.0.1:2379 \
  --advertise-client-urls http://127.0.0.1:2379 \
  --listen-peer-urls http://127.0.0.1:12380 \
  --initial-advertise-peer-urls http://127.0.0.1:12380 \
  --initial-cluster-token etcd-cluster-1 \
  --initial-cluster 'infra1=http://127.0.0.1:12380' \
  --initial-cluster-state new \
  --logger=zap --log-outputs=stderr
```

### 三节点（生产）

```bash
goreman -f Procfile start
```

---

## 验证

```bash
# 清除环境变量冲突
unset ETCDCTL_ENDPOINTS
unset ETCDCTL_API

# 检查健康
etcdctl --endpoints=localhost:2379 endpoint health

# 查看所有 key
etcdctl --endpoints=localhost:2379 get / --prefix
```

---

## 操作手册

### 配置热更新

```bash
# 更新 BTC RPC 地址
etcdctl --endpoints=localhost:2379 put \
  /blitz/config/btc_rpc_host \
  "localhost:18443/wallet/blitz_wallet"

# 更新 ETH RPC 地址
etcdctl --endpoints=localhost:2379 put \
  /blitz/config/eth_rpc_host \
  "http://localhost:8545"
```

服务日志应出现：
```
🔧 检测到 BTC_RPC_HOST 变更: ...
✅ BTC RPC 已热更新: ...
```

### 查看当前 Leader

```bash
etcdctl --endpoints=localhost:2379 get /blitz/leader/confirm-checker
```

### 查看活跃的提币锁

```bash
etcdctl --endpoints=localhost:2379 get /blitz/lock/ --prefix
```

### 手动释放卡住的锁（紧急情况）

```bash
etcdctl --endpoints=localhost:2379 del /blitz/lock/withdraw:user001:btc
```

---

## 多副本测试

```bash
# 终端1：正常端口
DATABASE_URL=... go run cmd/wallet-service/main.go

# 终端2：换端口模拟第二副本
DATABASE_URL=... PORT=2114 go run cmd/wallet-service/main.go
```

终端2 日志应看到：
```
🔎 ConfirmChecker 竞选失败，当前非 Leader，等待中...
```

杀掉终端1，终端2 在 15s 内（lease TTL）自动接管：
```
👑 ConfirmChecker 成为 Leader
```

---

## .gitignore

etcd 数据目录不提交：
```
*.etcd/
```
