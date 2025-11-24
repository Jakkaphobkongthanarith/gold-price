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
	Status         string                 `json:"status"`
	GoldTraders    *GoldPriceResponse     `json:"goldtraders"`
	InvestingCom   *InvestingGoldPrice    `json:"investing_com"`
	LastUpdate     string                 `json:"last_update"`
	mu             sync.RWMutex
	wsClients      map[*websocket.Conn]bool
	wsClientsMutex sync.Mutex
}

var (
	serverState = &ServerState{
		Status:       "online",
		wsClients:    make(map[*websocket.Conn]bool),
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
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serverState.mu.Lock()
	oldStatus := serverState.Status
	serverState.Status = req.Status
	serverState.mu.Unlock()

	if req.Status == "stopped" {
		resetPrices()
	} else if req.Status == "online" && oldStatus == "stopped" {
		// เมื่อกด ONLINE หลังจาก STOP → ดึงข้อมูลใหม่ทันที
		go func() {
			log.Println("🟢 System ONLINE - Fetching fresh data...")
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

	broadcastUpdate()

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": fmt.Sprintf("Status changed to %s", req.Status),
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
	
	log.Println("🛑 System STOPPED - All data cleared")
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

	// อนุญาตให้อัพเดทได้เฉพาะเมื่อ status เป็น online
	if serverState.Status != "online" {
		return
	}

	// อัพเดทข้อมูล
	if goldTraders != nil {
		serverState.GoldTraders = goldTraders
	}
	
	if investing != nil {
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

	fmt.Println("🌐 API Server started at " + config.ServerPort)
	fmt.Println("🖥️  Frontend available at http://localhost" + config.ServerPort)

	log.Fatal(http.ListenAndServe(config.ServerPort, nil))
}
