package types

// Chain 链类型
type Chain string

const (
	ChainBTC Chain = "btc"
	ChainETH Chain = "eth"
)

// AddressResponse 地址生成返回
type AddressResponse struct {
	Address string `json:"address"`
	Path    string `json:"path"`    // BIP44 路径
	UserID  string `json:"user_id"`
}

// BalanceResponse 余额返回
type BalanceResponse struct {
	Address string  `json:"address"`
	Balance float64 `json:"balance"`
	Chain   Chain   `json:"chain"`
}

// DepositRecord 充值记录
type DepositRecord struct {
	TxID      string  `json:"tx_id"`
	Address   string  `json:"address"`
	UserID    string  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Height    int64   `json:"height"`
	Confirmed bool    `json:"confirmed"`
	Chain     Chain   `json:"chain"`
}