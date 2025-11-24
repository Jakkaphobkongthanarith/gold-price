# 🥇 Gold Price Monitor - Real-time Gold Price Tracker

โปรเจกต์ระบบติดตามราคาทองคำแบบเรียลไทม์ จากหลายแหล่งข้อมูล พร้อม Dashboard แสดงผลและ REST API + WebSocket

---

## 📁 โครงสร้างโปรเจกต์

```
Gold/
├── 📂 backend/              # Backend Logic (ดึงข้อมูล + API Server)
│   ├── types.go            # Data structures และ types ทั้งหมด
│   ├── scraper.go          # ระบบดึงข้อมูลราคาทอง (Web Scraping)
│   └── api_server.go       # REST API + WebSocket Server
│
├── 📂 config/               # Configuration Settings
│   └── settings.go         # Constants, URLs, Intervals
│
├── 📂 frontend/             # Frontend Dashboard (แสดงผล)
│   └── index.html          # Dashboard UI (HTML/CSS/JS)
│
├── 📄 main.go               # จุดเริ่มต้นโปรแกรม (Entry Point)
├── 📄 go.mod                # Go modules dependencies
├── 📄 go.sum                # Go modules checksums
├── 📄 gold_prices.json      # ไฟล์เก็บข้อมูลราคา (Auto-generated)
└── 📄 README.md             # เอกสารนี้
```

---

## 🎯 หน้าที่ของแต่ละไฟล์

### 📂 **Backend** (หลังบ้าน - ดึงและจัดการข้อมูล)

#### **backend/types.go** - โครงสร้างข้อมูล

```go
// ประเภทข้อมูลที่ใช้ทั้งหมด:
- GoldPrice              // ราคาทองแต่ละประเภท (Buy/Sell)
- GoldPriceResponse      // ข้อมูลจาก GoldTraders
- InvestingGoldPrice     // ข้อมูลจาก Investing.com
- CombinedGoldData       // ข้อมูลรวมจากทุกแหล่ง
```

#### **backend/scraper.go** - ระบบดึงข้อมูล (Scraping)

```go
// ฟังก์ชันหลัก:
- MonitorInvestingCom()    // ตรวจสอบ Investing.com ทุก 2 วินาที
- MonitorGoldTraders()     // ตรวจสอบ GoldTraders ทุก 30 วินาที
- FetchInitialData()       // ดึงข้อมูลครั้งแรกตอนเริ่มโปรแกรม
- ParseGoldPrices()        // แปลง HTML เป็นข้อมูลราคา
- ParsePrice()             // แปลง string เป็น float64
```

**เทคนิคที่ใช้:**

- ✅ **Reusable Browser** - เปิด browser ครั้งเดียว ใช้ตลอด
- ✅ **chromedp** - Headless browser automation
- ✅ **Regex Parsing** - ดึงข้อมูลจาก HTML (GoldTraders)
- ✅ **CSS Selector** - ดึงข้อมูลจาก DOM (Investing.com)

#### **backend/api_server.go** - API + WebSocket Server

```go
// API Endpoints:
GET  /api/status           // ดึงสถานะและข้อมูลปัจจุบัน
POST /api/set-status       // เปลี่ยนสถานะ (online/paused/stopped)
WS   /ws                   // WebSocket สำหรับ real-time updates
GET  /                     // Serve frontend files
```

---

### 📂 **Config** (การตั้งค่า)

#### **config/settings.go** - Constants และ Settings

```go
InvestingComInterval = 2 * time.Second   // ตรวจสอบทุก 2 วินาที
GoldTradersInterval  = 10 * time.Second  // ตรวจสอบทุก 30 วินาที
ServerPort = ":8080"                     // พอร์ต API server
```

---

### 📂 **Frontend** (หน้าบ้าน - แสดงผล)

#### **frontend/index.html** - Dashboard UI

- 3 Price Cards (แสดงราคา)
- Transaction Table (ประวัติ)
- Control Buttons (Online/Pause/Stop)
- WebSocket Client (real-time updates)

---

### 📄 **main.go** - Entry Point

จุดเริ่มต้นโปรแกรม ทำหน้าที่:

1. เริ่ม API Server
2. ดึงข้อมูลครั้งแรก
3. เริ่ม monitoring goroutines
4. บันทึกข้อมูลลง JSON file

---

## 🚀 วิธีการรัน

### 1. Build โปรแกรม

```powershell
go build -o gold-monitor.exe
```

### 2. รันโปรแกรม

```powershell
.\gold-monitor.exe
```

### 3. เปิด Dashboard

เปิด browser ไปที่: **http://localhost:8080**

---

## 📊 Data Flow

```
Investing.com / GoldTraders
        ↓
backend/scraper.go (ดึงข้อมูล)
        ↓
main.go (บันทึก + ส่งต่อ)
        ↓
backend/api_server.go (Broadcast)
        ↓
frontend/index.html (แสดงผล)
```

---

## ⚙️ การปรับแต่ง

### เปลี่ยน Scraping Interval

แก้ไขใน `config/settings.go`:

```go
InvestingComInterval = 5 * time.Second
```

### เปลี่ยนพอร์ต

แก้ไขใน `config/settings.go`:

```go
ServerPort = ":3000"
```

---

## 🔧 เทคโนโลยีที่ใช้

- **Go 1.24** - Programming language
- **chromedp** - Browser automation
- **gorilla/websocket** - WebSocket
- **HTML/CSS/JavaScript** - Frontend

---

**ขอบคุณที่ใช้โปรเจกต์นี้!** 🙏
