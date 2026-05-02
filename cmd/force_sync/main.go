package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"database/sql"
	"ponches/internal/hikvision"
	"ponches/internal/store"
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

	// 1. Get managed devices
	var raw string
	err = db.QueryRow("SELECT value FROM app_config WHERE key = 'managed_devices'").Scan(&raw)
	if err != nil {
		log.Fatal("No managed devices found in DB:", err)
	}

	var devices []ManagedDevice
	json.Unmarshal([]byte(raw), &devices)

	// 2. Initialize store
	sqliteStore, err := store.NewSQLiteStore("./ponches.db")
	if err != nil {
		log.Fatal(err)
	}

	for _, d := range devices {
		fmt.Printf("--- Syncing Device: %s (%s) ---\n", d.Name, d.IP)
		
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
		
		events, err := client.GetEventsInRange(ctx, start, end)
		if err != nil {
			fmt.Printf("  [ERROR] Failed to fetch events: %v\n", err)
			continue
		}
		
		fmt.Printf("  [SUCCESS] Found %d events on device.\n", len(events))
		
		saved := 0
		for _, ev := range events {
			storeEvent := &store.AttendanceEvent{
				DeviceID:   ev.DeviceID,
				EmployeeNo: ev.EmployeeNo,
				Timestamp:  ev.Timestamp,
				Type:       ev.EventType,
			}
			err := sqliteStore.SaveEvent(ctx, storeEvent)
			if err != nil {
				fmt.Printf("    [ERROR] Save failed for emp %s at %v: %v\n", ev.EmployeeNo, ev.Timestamp, err)
			} else {
				saved++
			}
		}
		fmt.Printf("  [DONE] Saved %d events to DB.\n", saved)
	}
}
