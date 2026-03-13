package main

import "fmt"

func main() {
	eth := 32.0
	apr := 4.0 // 改成百分比形式，更直观（原来是 0.04）

	daily := eth * (apr / 100) / 365

	fmt.Printf("32 ETH stake APR %.1f%%: daily %.5f ETH\n", apr, daily)
	fmt.Printf("   → 年收益 ≈ %.2f ETH（%.1f%% APR）\n", eth*apr/100, apr)

	// Slash risk model（你注释里写的，真实 PoS 惩罚模拟）
	slashRisk := "线下 50% 概率 → 被 slashing 30%（扣 9.6 ETH）"
	fmt.Println("   ⚠️", slashRisk)
}
