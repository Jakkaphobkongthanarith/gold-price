package backend

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gold-scraper/config"

	"github.com/chromedp/chromedp"
)

func MonitorInvestingCom(ctx context.Context, onUpdate func(*InvestingGoldPrice)) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent(config.UserAgent),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	err := chromedp.Run(browserCtx,
		chromedp.Navigate(config.InvestingComURL),
	)
	if err != nil {
		log.Printf("⚠️  Cannot navigate to investing.com: %v", err)
		return
	}

	time.Sleep(config.InitialLoadTimeout)

	ticker := time.NewTicker(config.InvestingComInterval)
	defer ticker.Stop()

	retryCount := 0
	var lastPrice float64

	for {
		select {
		case <-ctx.Done():
			log.Println("✅ Investing.com monitor stopped")
			return
		case <-ticker.C:
		}

		var priceText, changeText, changePercentText string

		err := chromedp.Run(browserCtx,
			chromedp.Text(`[data-test="instrument-price-last"]`, &priceText, chromedp.ByQuery),
			chromedp.Text(`[data-test="instrument-price-change"]`, &changeText, chromedp.ByQuery, chromedp.NodeVisible),
			chromedp.Text(`[data-test="instrument-price-change-percent"]`, &changePercentText, chromedp.ByQuery, chromedp.NodeVisible),
		)

		if err != nil {
			retryCount++
			if retryCount >= config.MaxRetries {
				log.Printf("⚠️  Error fetching investing.com (retry %d/%d): %v", retryCount, config.MaxRetries, err)
				retryCount = 0
			}
			continue
		}

		retryCount = 0
		price := ParsePrice(priceText)

		if price != lastPrice && price > 0 {
			if lastPrice > 0 {
				fmt.Printf("\n🔔 [%s] Investing.com UPDATED!\n", time.Now().Format("15:04:05"))
				fmt.Printf("   Old: $%.2f → New: $%.2f (Change: $%.2f)\n",
					lastPrice, price, price-lastPrice)
			} else {
				fmt.Printf("\n💰 [%s] Investing.com: $%.2f USD\n", time.Now().Format("15:04:05"), price)
			}

			lastPrice = price

			newPrice := &InvestingGoldPrice{
				Type:          "Gold Spot Price (XAU/USD)",
				Price:         price,
				Change:        changeText,
				ChangePercent: changePercentText,
				UpdateTime:    time.Now().Format("2006-01-02 15:04:05"),
			}
			onUpdate(newPrice)
		}
	}
}

func MonitorGoldTraders(ctx context.Context, onUpdate func(*GoldPriceResponse, bool)) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.UserAgent(config.UserAgent),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	go func() {
		<-ctx.Done()
		log.Println("🛑 Closing GoldTraders browser...")
		browserCancel()
		allocCancel()
	}()

	err := chromedp.Run(browserCtx,
		chromedp.Navigate(config.GoldTradersURL),
	)
	if err != nil {
		log.Printf("⚠️  Cannot navigate to goldtraders: %v", err)
		return
	}

	time.Sleep(config.InitialLoadTimeout)

	ticker := time.NewTicker(config.GoldTradersInterval)
	defer ticker.Stop()

	var lastData *GoldPriceResponse

	for {
		select {
		case <-ctx.Done():
			log.Println("✅ GoldTraders monitor stopped")
			return
		case <-ticker.C:
		}

		var htmlContent string

		err := chromedp.Run(browserCtx,
			chromedp.OuterHTML("html", &htmlContent),
		)

		if err != nil {
			log.Printf("⚠️  Error fetching goldtraders: %v", err)
			continue
		}

		prices := ParseGoldPrices(htmlContent)

		if len(prices) == 0 {
			log.Printf("⚠️  No prices found from goldtraders")
			continue
		}

		newData := &GoldPriceResponse{
			Date:        time.Now().Format("2006-01-02"),
			LastUpdate:  time.Now().Format("2006-01-02 15:04:05"),
			Prices:      prices,
			Source:      config.GoldTradersURL,
			Description: "ราคาทองคำจากสมาคมค้าทองคำ (Reusable Browser)",
		}

		hasChanged := HasGoldTradersChanged(lastData, newData)
		
		if hasChanged {
			fmt.Printf("\n🔔 [%s] GoldTraders UPDATED!\n", time.Now().Format("15:04:05"))
			if lastData != nil {
				DisplayGoldTradersChanges(lastData, newData)
			} else {
				DisplayGoldTradersPrices(newData)
			}
		} else {
			fmt.Printf("   [%s] GoldTraders: No change\n", time.Now().Format("15:04:05"))
		}

		lastData = newData
		onUpdate(newData, hasChanged)
	}
}

func FetchInitialData() (*GoldPriceResponse, *InvestingGoldPrice) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
	)

	var goldTraders *GoldPriceResponse
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var htmlContent string
	err := chromedp.Run(browserCtx,
		chromedp.Navigate(config.GoldTradersURL),
		chromedp.Sleep(3*time.Second),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err == nil {
		prices := ParseGoldPrices(htmlContent)
		if len(prices) > 0 {
			goldTraders = &GoldPriceResponse{
				Date:        time.Now().Format("2006-01-02"),
				LastUpdate:  time.Now().Format("2006-01-02 15:04:05"),
				Prices:      prices,
				Source:      config.GoldTradersURL,
				Description: "ราคาทองคำจากสมาคมค้าทองคำ",
			}
			DisplayGoldTradersPrices(goldTraders)
		}
	} else {
		log.Printf("⚠️  Could not fetch initial goldtraders data: %v", err)
	}

	var investing *InvestingGoldPrice
	allocCtx2, cancel2 := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel2()
	browserCtx2, cancel2 := chromedp.NewContext(allocCtx2)
	defer cancel2()

	var priceText string
	err2 := chromedp.Run(browserCtx2,
		chromedp.Navigate(config.InvestingComURL),
		chromedp.Sleep(3*time.Second),
		chromedp.Text(`[data-test="instrument-price-last"]`, &priceText, chromedp.ByQuery),
	)

	if err2 == nil {
		price := ParsePrice(priceText)
		if price > 0 {
			fmt.Printf("💰 Investing.com: $%.2f USD\n", price)
			investing = &InvestingGoldPrice{
				Type:       "Gold Spot Price (XAU/USD)",
				Price:      price,
				UpdateTime: time.Now().Format("2006-01-02 15:04:05"),
			}
		}
	} else {
		log.Printf("⚠️  Could not fetch initial investing data: %v", err2)
	}

	fmt.Println()
	return goldTraders, investing
}

func ParseGoldPrices(htmlContent string) []GoldPrice {
	var prices []GoldPrice
	currentTime := time.Now().Format("2006-01-02 15:04:05")

	lblBLSellPattern := regexp.MustCompile(`id="DetailPlace_uc_goldprices1_lblBLSell"[^>]*>(.*?)([\d,]+\.?\d*)</font>`)
	if matches := lblBLSellPattern.FindStringSubmatch(htmlContent); len(matches) >= 3 {
		sellBar := ParsePrice(matches[2])

		lblBLBuyPattern := regexp.MustCompile(`id="DetailPlace_uc_goldprices1_lblBLBuy"[^>]*>(.*?)([\d,]+\.?\d*)</font>`)
		if matchesBuy := lblBLBuyPattern.FindStringSubmatch(htmlContent); len(matchesBuy) >= 3 {
			buyBar := ParsePrice(matchesBuy[2])

			if buyBar > 0 && sellBar > 0 {
				prices = append(prices, GoldPrice{
					Type:       "ทองคำแท่ง 96.5% (Gold Bar)",
					BuyPrice:   buyBar,
					SellPrice:  sellBar,
					UpdateTime: currentTime,
				})
			}
		}
	}

	lblOMSellPattern := regexp.MustCompile(`id="DetailPlace_uc_goldprices1_lblOMSell"[^>]*>(.*?)([\d,]+\.?\d*)</font>`)
	if matches := lblOMSellPattern.FindStringSubmatch(htmlContent); len(matches) >= 3 {
		sellJewelry := ParsePrice(matches[2])

		lblOMBuyPattern := regexp.MustCompile(`id="DetailPlace_uc_goldprices1_lblOMBuy"[^>]*>(.*?)([\d,]+\.?\d*)</font>`)
		if matchesBuy := lblOMBuyPattern.FindStringSubmatch(htmlContent); len(matchesBuy) >= 3 {
			buyJewelry := ParsePrice(matchesBuy[2])

			if buyJewelry > 0 && sellJewelry > 0 {
				prices = append(prices, GoldPrice{
					Type:       "ทองรูปพรรณ 96.5% (Jewelry Gold)",
					BuyPrice:   buyJewelry,
					SellPrice:  sellJewelry,
					UpdateTime: currentTime,
				})
			}
		}
	}

	return prices
}

func ParsePrice(s string) float64 {
	cleaned := strings.ReplaceAll(s, ",", "")
	cleaned = strings.TrimSpace(cleaned)
	price, _ := strconv.ParseFloat(cleaned, 64)
	return price
}

func HasGoldTradersChanged(old, new *GoldPriceResponse) bool {
	if old == nil || new == nil {
		return true
	}

	if len(old.Prices) != len(new.Prices) {
		return true
	}

	for i := range old.Prices {
		if old.Prices[i].BuyPrice != new.Prices[i].BuyPrice ||
			old.Prices[i].SellPrice != new.Prices[i].SellPrice {
			return true
		}
	}

	return false
}

func DisplayGoldTradersChanges(old, new *GoldPriceResponse) {
	for i := range new.Prices {
		if i < len(old.Prices) {
			oldBuy := old.Prices[i].BuyPrice
			oldSell := old.Prices[i].SellPrice
			newBuy := new.Prices[i].BuyPrice
			newSell := new.Prices[i].SellPrice

			if oldBuy != newBuy || oldSell != newSell {
				fmt.Printf("   %s:\n", new.Prices[i].Type)
				fmt.Printf("      Buy:  %.2f → %.2f (%.2f)\n", oldBuy, newBuy, newBuy-oldBuy)
				fmt.Printf("      Sell: %.2f → %.2f (%.2f)\n", oldSell, newSell, newSell-oldSell)
			}
		}
	}
}

func DisplayGoldTradersPrices(data *GoldPriceResponse) {
	for _, price := range data.Prices {
		fmt.Printf("   %s: Buy %.2f | Sell %.2f\n", price.Type, price.BuyPrice, price.SellPrice)
	}
}
