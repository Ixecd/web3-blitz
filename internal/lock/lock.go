package lock

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type DistributedLock struct {
	client *clientv3.Client
	ttl    int64
}

func NewDistributedLock(client *clientv3.Client, ttl int64) *DistributedLock {
	return &DistributedLock{client: client, ttl: ttl}
}

type Lock struct {
	leaseID clientv3.LeaseID
	key     string
	client  *clientv3.Client
}

// Acquire 尝试获取锁，成功返回 Lock，失败返回 error
func (d *DistributedLock) Acquire(ctx context.Context, key string) (*Lock, error) {
	// 1. 创建 lease（TTL 秒后自动过期，进程崩溃也不会死锁）
	lease, err := d.client.Grant(ctx, d.ttl)
	if err != nil {
		return nil, fmt.Errorf("创建 lease 失败: %w", err)
	}

	fullKey := fmt.Sprintf("/blitz/lock/%s", key)

	// 2. 用事务原子性加锁：key 不存在才写入
	txn := d.client.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(fullKey), "=", 0)).
		Then(clientv3.OpPut(fullKey, "1", clientv3.WithLease(lease.ID))).
		Else()

	resp, err := txn.Commit()
	if err != nil {
		d.client.Revoke(ctx, lease.ID)
		return nil, fmt.Errorf("加锁事务失败: %w", err)
	}

	if !resp.Succeeded {
		// key 已存在，锁被占用
		d.client.Revoke(ctx, lease.ID)
		return nil, fmt.Errorf("锁已被占用，请勿重复提交")
	}

	return &Lock{leaseID: lease.ID, key: fullKey, client: d.client}, nil
}

// Release 释放锁（撤销 lease，key 自动删除）
func (l *Lock) Release(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	l.client.Revoke(ctx, l.leaseID)
}