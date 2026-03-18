# etcd 架构设计

> web3-blitz 中 etcd 的三个使用场景及设计思路。

---

## 为什么选 etcd

交易所钱包服务是资金敏感系统，核心诉求是**正确性优先于可用性**：

- 提币不能重复广播（资金损失）
- 充值确认不能重复执行（数据重复）
- 配置变更不能导致服务重启（影响充提币监听）

etcd 基于 Raft 协议，提供强一致性的分布式协调能力，天然适合这三类场景。相比 Redis，etcd 的 lease 机制在进程崩溃时能自动释放锁，不会出现死锁。

---

## 场景一：分布式锁（防重复提币）

### 问题

K8s 多副本部署时，同一用户的两个并发提币请求可能同时通过余额校验，然后各自广播一笔交易，造成资金双花。

### 方案

用 etcd lease + 事务 CAS 实现分布式锁：

```
用户发起提币
    │
    ▼
尝试获取锁（lease + txn CAS）
    ├─ 成功 → 继续余额校验 → 广播 → 释放锁
    └─ 失败 → 返回 429 请勿重复提交
```

### 锁 key 设计

```
/blitz/lock/withdraw:{user_id}:{chain}
```

按用户 + 链维度加锁，不同用户、不同链的提币互不影响。

### 核心代码

```go
// 创建 lease（TTL 30s，进程崩溃自动释放）
lease, _ := client.Grant(ctx, 30)

// 原子竞争：key 不存在才写入
txn := client.Txn(ctx).
    If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
    Then(clientv3.OpPut(key, "1", clientv3.WithLease(lease.ID))).
    Else()

resp, _ := txn.Commit()
if !resp.Succeeded {
    // 锁已被占用
}
```

### 为什么用 lease 而不是 SET NX

Redis SET NX 存在网络分区时锁不释放的风险。etcd lease 与 etcd server 保持心跳，server 挂了 lease 自动过期，进程崩溃同理。

---

## 场景二：选主（ConfirmChecker 单活）

### 问题

`ConfirmChecker` 每 30 秒扫一次未确认充值并更新状态。多副本部署时所有实例都在跑，会重复执行 `UPDATE deposits SET confirmed=1`，虽然幂等但浪费资源，且日志混乱。

### 方案

用 etcd lease + KeepAlive 实现选主，只有 Leader 跑 ConfirmChecker：

```
副本启动 → campaign 竞选
    ├─ 抢到锁 → 成为 Leader → KeepAlive 续期 → 每30s执行 check
    └─ 未抢到 → 等 3s 重试
            ↑
Leader 挂掉 → lease TTL(15s) 过期 → 其他副本抢到锁 → 成为新 Leader
```

### 选主 key

```
/blitz/leader/confirm-checker
```

### Failover 时间

`leaseTTL = 15s`，Leader 崩溃后最多 15 秒内完成切换。对于 30 秒轮询周期的 ConfirmChecker，这个延迟完全可以接受。

### 正常退出 vs 崩溃

- 正常退出（Ctrl+C）：`defer Revoke()` 立即释放 lease，其他副本几乎立刻接管
- 进程崩溃：lease TTL 过期后自动释放，最多等待 15s

---

## 场景三：配置热更新

### 问题

`BTC_RPC_HOST` 和 `ETH_RPC_HOST` 写死在环境变量里，bitcoind 或 geth 地址变更时必须重启服务，会中断充值监听。

### 方案

用 etcd watch 监听配置变更，重建 RPC 连接后原子替换：

```
运维执行 etcdctl put /blitz/config/btc_rpc_host <新地址>
    │
    ▼
ConfigWatcher 收到 watch 事件
    │
    ▼
重建 rpcclient.Client
    │
    ▼
BTCRPCHolder.Set(newRPC)  ← 原子替换，读写锁保护
    │
    ▼
所有组件下次调用 holder.Get() 时自动用新连接
```

### config key

```
/blitz/config/btc_rpc_host
/blitz/config/eth_rpc_host
```

### RPC Holder 设计

所有组件（BTCWallet、DepositWatcher、ConfirmChecker）不直接持有 `*rpcclient.Client`，而是持有 `*BTCRPCHolder`，通过 `Get()` 获取当前客户端：

```go
type BTCRPCHolder struct {
    mu  sync.RWMutex
    rpc *rpcclient.Client
}

func (h *BTCRPCHolder) Get() *rpcclient.Client {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return h.rpc
}

func (h *BTCRPCHolder) Set(rpc *rpcclient.Client) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.rpc = rpc
}
```

热更新对所有调用方完全透明，无需改动业务代码。

### 触发热更新

```bash
etcdctl --endpoints=localhost:2379 put \
  /blitz/config/btc_rpc_host \
  "localhost:18443/wallet/blitz_wallet"
```

---

## etcd key 总览

| key | 用途 | TTL |
|-----|------|-----|
| `/blitz/lock/withdraw:{uid}:{chain}` | 提币分布式锁 | 30s |
| `/blitz/leader/confirm-checker` | ConfirmChecker 选主 | 15s |
| `/blitz/config/btc_rpc_host` | BTC RPC 地址 | 永久 |
| `/blitz/config/eth_rpc_host` | ETH RPC 地址 | 永久 |

---

## 本地启动

```bash
# 单节点（开发用）
goreman -f Procfile.single start

# 验证
etcdctl --endpoints=localhost:2379 endpoint health
```

生产环境需要三节点集群，参考 `Procfile`。
