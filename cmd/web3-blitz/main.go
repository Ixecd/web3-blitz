package main

import (
	"log"

	// 必须保留，用于地址类型
	"github.com/btcsuite/btcd/rpcclient"
)

func main() {
	cfg := &rpcclient.ConnConfig{
		Host:         "localhost:18443",
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	client, err := rpcclient.New(cfg, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Shutdown()

	// === 1. 查询余额 ===
	bal, err := client.GetBalance("*")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Balance BTC:", bal.ToBTC())

	// === 2. 新版挖块（关键修复）===
	addr, err := client.GetNewAddress("")
	if err != nil {
		log.Fatal(err)
	}

	// 第三个参数 nil = 使用默认最大尝试次数
	hashes, err := client.GenerateToAddress(5, addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("New blocks generated:", hashes)
}
