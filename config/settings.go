package config

import "time"

const (
	InvestingComInterval = 2 * time.Second
	GoldTradersInterval  = 10 * time.Second
	
	InitialLoadTimeout = 3 * time.Second
	
	MaxRetries = 3
)

const (
	ServerPort = ":8080"
	DataFile   = "gold_prices.json"
)

const (
	InvestingComURL = "https://th.investing.com/commodities/gold"
	GoldTradersURL  = "https://www.goldtraders.or.th/default.aspx"
)

const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
