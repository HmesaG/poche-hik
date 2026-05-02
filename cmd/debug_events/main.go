package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"database/sql"
	"ponches/internal/hikvision"
	_ "modernc.org/sqlite"
)

type ManagedDevice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Port      int    `json:"port"`
	TimezoneOffset string `json:"timezoneOffset"`
}

func main() {
	db, err := sql.Open("sqlite", `./ponches.db`)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var raw string
	err = db.QueryRow("SELECT value FROM app_config WHERE key = 'managed_devices'").Scan(&raw)
	if err != nil {
		log.Fatal("No managed devices found in DB:", err)
	}

	var devices []ManagedDevice
	if err := json.Unmarshal([]byte(raw), &devices); err != nil {
		log.Fatal(err)
	}

	for _, d := range devices {
		fmt.Printf("--- Checking Device: %s (%s) ---\n", d.Name, d.IP)
		
		client := hikvision.NewClient(d.IP, d.Port, d.Username, d.Password)
		client.TimezoneOffset = d.TimezoneOffset
		if client.TimezoneOffset == "" {
			client.TimezoneOffset = "+08:00"
		}
		
		ctx := context.Background()
		
		// Range: From 2026-04-01 to now
		now := time.Now()
		start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
		end := now.Add(1 * time.Hour)
		
		fmt.Printf("Fetching events from %s to %s\n", start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))
		
		events, err := client.GetEventsInRange(ctx, start, end)
		if err != nil {
			fmt.Printf("  [ERROR] Failed to fetch events: %v\n", err)
			continue
		}
		
		fmt.Printf("  [SUCCESS] Found %d events on device.\n", len(events))
		
		if len(events) > 0 {
			fmt.Println("  Sample events (last 5):")
			startIdx := len(events) - 5
			if startIdx < 0 { startIdx = 0 }
			for i := startIdx; i < len(events); i++ {
				ev := events[i]
				fmt.Printf("    - Employee: %s | Time: %s | Type: %s\n", ev.EmployeeNo, ev.Timestamp.Format("2006-01-02 15:04:05"), ev.EventType)
			}
		} else {
			fmt.Println("  [WARN] Device returned ZERO events for this range.")
		}
	}
}
