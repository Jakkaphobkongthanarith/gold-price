package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gold-scraper/backend"
	"gold-scraper/config"
)

const TransactionsFile = "transactions.json"

func main() {
	fmt.Println("🚀 Starting Real-time Gold Price Monitor with API Server")
	fmt.Println("📊 Monitoring changes from multiple sources...")
	fmt.Println("⏱️  Investing.com: Check every 2 seconds (Reusable Browser)")
	fmt.Println("⏱️  GoldTraders: Check every 30 seconds (Reusable Browser)")
	fmt.Println("💡 Press Ctrl+C to stop")
	fmt.Println(strings.Repeat("=", 70))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("interrupt signal")
		cancel()
	}()

	go backend.StartAPIServer()

	time.Sleep(2 * time.Second)

	fmt.Println("\n Initial fetch...")
	goldTraders, investing := backend.FetchInitialData()
	
	if goldTraders != nil {
		saveGoldTradersData(goldTraders)
		if len(goldTraders.Prices) > 0 {
			saveTransaction("ทองแท่ง Buy", goldTraders.Prices[0].BuyPrice, "buy")
			saveTransaction("ทองแท่ง Sell", goldTraders.Prices[0].SellPrice, "sell")
			if len(goldTraders.Prices) > 1 {
				saveTransaction("ทองรูปพรรณ Buy", goldTraders.Prices[1].BuyPrice, "buy")
				saveTransaction("ทองรูปพรรณ Sell", goldTraders.Prices[1].SellPrice, "sell")
			}
		}
		backend.UpdateServerData(goldTraders, nil)
	}
	
	if investing != nil {
		saveInvestingData(investing)
		saveTransaction("Investing.com (XAU/USD)", investing.Price, "market")
		backend.UpdateServerData(nil, investing)
	}

	go backend.MonitorInvestingCom(ctx, func(data *backend.InvestingGoldPrice) {
		saveInvestingData(data)
		saveTransaction("Investing.com (XAU/USD)", data.Price, "market")
		backend.UpdateServerData(nil, data)
	})

	go backend.MonitorGoldTraders(ctx, func(data *backend.GoldPriceResponse, hasChanged bool) {
		if hasChanged {
			saveGoldTradersData(data)
			if len(data.Prices) > 0 {
				saveTransaction("ทองแท่ง Buy", data.Prices[0].BuyPrice, "buy")
				saveTransaction("ทองแท่ง Sell", data.Prices[0].SellPrice, "sell")
				if len(data.Prices) > 1 {
					saveTransaction("ทองรูปพรรณ Buy", data.Prices[1].BuyPrice, "buy")
					saveTransaction("ทองรูปพรรณ Sell", data.Prices[1].SellPrice, "sell")
				}
			}
			backend.UpdateServerData(data, nil)
		}
	})

	<-ctx.Done()
	log.Println("Cleaning up Chrome processes...")
	time.Sleep(1 * time.Second)
	log.Println("Shutdown")
}

func saveTransaction(symbol string, price float64, state string) {
	tx := backend.Transaction{
		Symbol:   symbol,
		Price:    price,
		State:    state,
		DateTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	transactions := loadTransactions()

	transactions = append([]backend.Transaction{tx}, transactions...)

	if len(transactions) > 1000 {
		transactions = transactions[:1000]
	}

	file, err := os.Create(TransactionsFile)
	if err != nil {
		log.Printf("Error saving transaction: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(transactions); err != nil {
		log.Printf("⚠️  Error encoding transactions: %v", err)
	}
}

func loadTransactions() []backend.Transaction {
	file, err := os.Open(TransactionsFile)
	if err != nil {
		return []backend.Transaction{}
	}
	defer file.Close()

	var transactions []backend.Transaction
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&transactions); err != nil {
		return []backend.Transaction{}
	}
	return transactions
}


func saveInvestingData(data *backend.InvestingGoldPrice) {
	combinedData := loadCombinedData()
	combinedData.InvestingCom = data
	combinedData.Date = time.Now().Format("2006-01-02")
	combinedData.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
	saveCombinedData(combinedData)
}

func saveGoldTradersData(data *backend.GoldPriceResponse) {
	combinedData := loadCombinedData()
	combinedData.GoldTraders = data
	combinedData.Date = time.Now().Format("2006-01-02")
	combinedData.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
	saveCombinedData(combinedData)
}

func loadCombinedData() *backend.CombinedGoldData {
	file, err := os.Open(config.DataFile)
	if err != nil {
		return &backend.CombinedGoldData{}
	}
	defer file.Close()

	var data backend.CombinedGoldData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return &backend.CombinedGoldData{}
	}
	return &data
}

func saveCombinedData(data *backend.CombinedGoldData) {
	file, err := os.Create(config.DataFile)
	if err != nil {
		log.Printf("⚠️  Error saving data: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(data)
}
