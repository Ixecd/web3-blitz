package main

import (
	"log"

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

	bal, err := client.GetBalance("*")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Balance BTC:", bal.ToBTC())

	hashes, err := client.Generate(5)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("New blocks:", hashes)
}
