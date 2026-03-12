package main

import "fmt"

func main() {
	eth := 32.0
	apr := 0.04  // 4%
	daily := eth * apr / 365
	fmt.Printf("32 ETH stake APR %.1f%%: daily %.2f ETH\n", apr*100, daily)
	// Slash risk model: offline 50% = slash 30%
}