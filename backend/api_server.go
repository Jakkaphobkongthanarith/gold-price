package backend

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"gold-scraper/config"

	"github.com/gorilla/websocket"
)

type ServerState struct {
	Status              string                 `json:"status"`
	InvestingStatus     string                 `json:"investing_status"`
	GoldTradersStatus   string                 `json:"goldtraders_status"`
	GoldBarStatus       string                 `json:"goldbar_status"`
	GoldJewelryStatus   string                 `json:"goldjewelry_status"`
	GoldTraders         *GoldPriceResponse     `json:"goldtraders"`
	InvestingCom        *InvestingGoldPrice    `json:"investing_com"`
	LastUpdate          string                 `json:"last_update"`
	mu                  sync.RWMutex
	wsClients           map[*websocket.Conn]bool
	wsClientsMutex      sync.Mutex
}

var (
	serverState = &ServerState{
		Status:            "online",
		InvestingStatus:   "online",
		GoldTradersStatus: "online",
		GoldBarStatus:     "online",
		GoldJewelryStatus: "online",
		wsClients:         make(map[*websocket.Conn]bool),
	}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// API Handlers
func handleGetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	serverState.mu.RLock()
	defer serverState.mu.RUnlock()

	json.NewEncoder(w).Encode(serverState)
}

func handleSetStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req struct {
		Status string `json:"status"`
		Source string `json:"source"` // "all", "investing", "goldtraders"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serverState.mu.Lock()
	
	switch req.Source {
	case "investing":
		oldStatus := serverState.InvestingStatus
		serverState.InvestingStatus = req.Status
		
		if req.Status == "stopped" {
			serverState.InvestingCom = nil
			log.Println("STOPPED")
		} else if req.Status == "online" && oldStatus == "stopped" {
			go func() {
				log.Println("ONLINE")
				_, investing := FetchInitialData()
				if investing != nil {
					serverState.mu.Lock()
					serverState.InvestingCom = investing
					serverState.mu.Unlock()
					broadcastUpdate()
				}
			}()
		}
		
	case "goldtraders":
		oldStatus := serverState.GoldTradersStatus
		serverState.GoldTradersStatus = req.Status
		serverState.GoldBarStatus = req.Status
		serverState.GoldJewelryStatus = req.Status
		
		if req.Status == "stopped" {
			serverState.GoldTraders = nil
			log.Println("STOPPED")
		} else if req.Status == "online" && oldStatus == "stopped" {
			go func() {
				log.Println("ONLINE")
				goldTraders, _ := FetchInitialData()
				if goldTraders != nil {
					serverState.mu.Lock()
					serverState.GoldTraders = goldTraders
					serverState.mu.Unlock()
					broadcastUpdate()
				}
			}()
		}
		
	case "goldbar":
		serverState.GoldBarStatus = req.Status
		
	case "goldjewelry":
		serverState.GoldJewelryStatus = req.Status
		
	default: // "all"
		oldStatus := serverState.Status
		serverState.Status = req.Status
		serverState.InvestingStatus = req.Status
		serverState.GoldTradersStatus = req.Status
		serverState.GoldBarStatus = req.Status
		serverState.GoldJewelryStatus = req.Status
		
		if req.Status == "stopped" {
			serverState.GoldTraders = nil
			serverState.InvestingCom = nil
			log.Println("STOPPED")
		} else if req.Status == "online" && oldStatus == "stopped" {
			go func() {
				log.Println("ONLINE")
				goldTraders, investing := FetchInitialData()
				
				serverState.mu.Lock()
				if goldTraders != nil {
					serverState.GoldTraders = goldTraders
				}
				if investing != nil {
					serverState.InvestingCom = investing
				}
				serverState.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
				serverState.mu.Unlock()
				
				broadcastUpdate()
			}()
		}
	}
	
	serverState.mu.Unlock()
	broadcastUpdate()

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Status changed to %s for %s", req.Status, req.Source),
	})
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	serverState.wsClientsMutex.Lock()
	serverState.wsClients[conn] = true
	serverState.wsClientsMutex.Unlock()

	serverState.mu.RLock()
	conn.WriteJSON(serverState)
	serverState.mu.RUnlock()

	go func() {
		defer func() {
			serverState.wsClientsMutex.Lock()
			delete(serverState.wsClients, conn)
			serverState.wsClientsMutex.Unlock()
			conn.Close()
		}()

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}

func resetPrices() {
	serverState.mu.Lock()
	defer serverState.mu.Unlock()

	// เคลียร์ข้อมูลทั้งหมดเป็น nil
	serverState.GoldTraders = nil
	serverState.InvestingCom = nil
	serverState.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
	
	log.Println("STOPPED")
}

func broadcastUpdate() {
	serverState.wsClientsMutex.Lock()
	defer serverState.wsClientsMutex.Unlock()

	serverState.mu.RLock()
	data := serverState
	serverState.mu.RUnlock()

	for client := range serverState.wsClients {
		err := client.WriteJSON(data)
		if err != nil {
			client.Close()
			delete(serverState.wsClients, client)
		}
	}
}

func UpdateServerData(goldTraders *GoldPriceResponse, investing *InvestingGoldPrice) {
	serverState.mu.Lock()
	defer serverState.mu.Unlock()

	// อัพเดทข้อมูลตาม status ของแต่ละแหล่ง
	if goldTraders != nil && serverState.GoldTradersStatus == "online" {
		serverState.GoldTraders = goldTraders
	}
	
	if investing != nil && serverState.InvestingStatus == "online" {
		serverState.InvestingCom = investing
	}
	
	serverState.LastUpdate = time.Now().Format("2006-01-02 15:04:05")

	go broadcastUpdate()
}

func StartAPIServer() {
	http.HandleFunc("/api/status", handleGetStatus)
	http.HandleFunc("/api/set-status", handleSetStatus)
	http.HandleFunc("/ws", handleWebSocket)
	http.Handle("/", http.FileServer(http.Dir("./frontend")))

	fmt.Println("API Server " + config.ServerPort)
	fmt.Println("Frontend: http://localhost" + config.ServerPort)

	log.Fatal(http.ListenAndServe(config.ServerPort, nil))
}
