package main

import (
	"log"
	"os"

	"github.com/btcsuite/btcd/rpcclient"
)

func main() {
	btcRPCHost := os.Getenv("BTC_RPC_HOST")
	if btcRPCHost == "" {
		btcRPCHost = "localhost:18443"
	}

	cfg := &rpcclient.ConnConfig{
		Host:         btcRPCHost,
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

	bal, err := client.GetBalance("*")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Balance BTC:", bal.ToBTC())

	addr, err := client.GetNewAddress("")
	if err != nil {
		log.Fatal(err)
	}

	hashes, err := client.GenerateToAddress(5, addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("New blocks generated:", hashes)
}