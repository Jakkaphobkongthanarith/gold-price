package backend

type GoldPrice struct {
	Type       string  `json:"type"`
	BuyPrice   float64 `json:"buy_price"`
	SellPrice  float64 `json:"sell_price"`
	UpdateTime string  `json:"update_time"`
}

type GoldPriceResponse struct {
	Date        string      `json:"date"`
	LastUpdate  string      `json:"last_update"`
	Prices      []GoldPrice `json:"prices"`
	Source      string      `json:"source"`
	Description string      `json:"description"`
}

type InvestingGoldPrice struct {
	Type          string  `json:"type"`
	Price         float64 `json:"price"`
	Change        string  `json:"change"`
	ChangePercent string  `json:"change_percent"`
	Currency      string  `json:"currency"`
	UpdateTime    string  `json:"update_time"`
}

type CombinedGoldData struct {
	Date         string              `json:"date"`
	LastUpdate   string              `json:"last_update"`
	GoldTraders  *GoldPriceResponse  `json:"goldtraders"`
	InvestingCom *InvestingGoldPrice `json:"investing_com"`
}
