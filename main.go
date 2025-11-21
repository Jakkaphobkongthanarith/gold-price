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
		log.Println("\n🛑 Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	go backend.StartAPIServer()

	time.Sleep(2 * time.Second)

	fmt.Println("\n🔄 Initial fetch...")
	goldTraders, investing := backend.FetchInitialData()
	
	if goldTraders != nil {
		saveGoldTradersData(goldTraders)
		backend.UpdateServerData(goldTraders, nil)
	}
	
	if investing != nil {
		saveInvestingData(investing)
		backend.UpdateServerData(nil, investing)
	}

	go backend.MonitorInvestingCom(ctx, func(data *backend.InvestingGoldPrice) {
		saveInvestingData(data)
		backend.UpdateServerData(nil, data)
	})

	go backend.MonitorGoldTraders(ctx, func(data *backend.GoldPriceResponse, hasChanged bool) {
		if hasChanged {
			saveGoldTradersData(data)
			backend.UpdateServerData(data, nil)
		}
	})

	<-ctx.Done()
	log.Println("✅ Cleaning up Chrome processes...")
	time.Sleep(1 * time.Second)
	log.Println("✅ Shutdown complete. Goodbye!")
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
