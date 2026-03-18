package core

import (
	"context"
	"log"
	"time"

	"github.com/Ixecd/web3-blitz/internal/config"
	"github.com/Ixecd/web3-blitz/internal/db"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	BTCRequiredConfirms = 6
	ETHRequiredConfirms = 12
	leaderKey           = "/blitz/leader/confirm-checker"
	leaseTTL            = 15 // 秒
)

type ConfirmChecker struct {
	queries    *db.Queries
	btcRPC     *config.BTCRPCHolder
	ethRPC     *config.ETHRPCHolder
	etcdClient *clientv3.Client
}

func NewConfirmChecker(queries *db.Queries, btcRPC *config.BTCRPCHolder, ethRPC *config.ETHRPCHolder, etcdClient *clientv3.Client) *ConfirmChecker {
	return &ConfirmChecker{
		queries:    queries,
		btcRPC:     btcRPC,
		ethRPC:     ethRPC,
		etcdClient: etcdClient,
	}
}

func (c *ConfirmChecker) Start(ctx context.Context) {
	log.Println("🔎 ConfirmChecker 已启动，竞选 Leader...")

	for {
		select {
		case <-ctx.Done():
			log.Println("⛔ ConfirmChecker 已停止")
			return
		default:
			c.campaign(ctx)
			// campaign 退出说明失去 Leader，等 3s 再重新竞选
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}
}

// campaign 竞选 Leader，成功后持续持有 lease 并执行 check
// lease 过期或 ctx 取消时退出
func (c *ConfirmChecker) campaign(ctx context.Context) {
	// 1. 创建 lease
	lease, err := c.etcdClient.Grant(ctx, leaseTTL)
	if err != nil {
		log.Printf("[WARN] ConfirmChecker 创建 lease 失败: %v", err)
		return
	}

	// 2. 原子竞选：key 不存在才写入
	txn := c.etcdClient.Txn(ctx).
		If(clientv3.Compare(clientv3.CreateRevision(leaderKey), "=", 0)).
		Then(clientv3.OpPut(leaderKey, "1", clientv3.WithLease(lease.ID))).
		Else()

	resp, err := txn.Commit()
	if err != nil || !resp.Succeeded {
		// 竞选失败，撤销 lease，等待重试
		c.etcdClient.Revoke(ctx, lease.ID)
		if err == nil {
			log.Println("🔎 ConfirmChecker 竞选失败，当前非 Leader，等待中...")
		}
		return
	}

	log.Println("👑 ConfirmChecker 成为 Leader")

	// 3. 启动 lease 续期（keepalive），保持 Leader 身份
	keepAlive, err := c.etcdClient.KeepAlive(ctx, lease.ID)
	if err != nil {
		c.etcdClient.Revoke(ctx, lease.ID)
		log.Printf("[WARN] ConfirmChecker keepalive 失败: %v", err)
		return
	}

	// 4. 作为 Leader 跑 check 循环
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer c.etcdClient.Revoke(context.Background(), lease.ID)

	for {
		select {
		case <-ctx.Done():
			log.Println("⛔ ConfirmChecker Leader 退出")
			return
		case _, ok := <-keepAlive:
			if !ok {
				// keepalive channel 关闭，lease 过期，失去 Leader
				log.Println("⚠️  ConfirmChecker lease 过期，重新竞选")
				return
			}
		case <-ticker.C:
			c.check(ctx)
		}
	}
}

func (c *ConfirmChecker) check(ctx context.Context) {
	deposits, err := c.queries.ListUnconfirmedDeposits(ctx)
	if err != nil {
		log.Printf("[ERROR] ConfirmChecker 查询未确认充值失败: %v", err)
		return
	}
	if len(deposits) == 0 {
		return
	}

	var btcHeight int64
	if info, err := c.btcRPC.Get().GetBlockChainInfo(); err == nil {
		btcHeight = int64(info.Blocks)
	}

	var ethHeight int64
	if header, err := c.ethRPC.Get().HeaderByNumber(ctx, nil); err == nil {
		ethHeight = header.Number.Int64()
	}

	for _, d := range deposits {
		var required int64
		var currentHeight int64

		switch d.Chain {
		case "btc":
			required = BTCRequiredConfirms
			currentHeight = btcHeight
		case "eth":
			required = ETHRequiredConfirms
			currentHeight = ethHeight
		default:
			continue
		}

		if currentHeight-d.Height >= required {
			if err := c.queries.UpdateDepositConfirmed(ctx, d.ID); err != nil {
				log.Printf("[ERROR] 更新confirmed失败 id=%d: %v", d.ID, err)
				continue
			}
			log.Printf("✅ 充值已确认 id=%d chain=%s txid=%s (块高差=%d)",
				d.ID, d.Chain, d.TxID, currentHeight-d.Height)
		}
	}
}
