package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"ponches/internal/store"
	_ "modernc.org/sqlite"
)

func main() {
	sqliteStore, err := store.NewSQLiteStore("./ponches.db")
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	to := from.Add(24 * time.Hour)

	fmt.Printf("Searching from %v to %v\n", from, to)

	events, err := sqliteStore.GetEvents(context.Background(), store.EventFilter{
		From: from,
		To:   to,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d events via GetEvents.\n", len(events))
	for _, ev := range events {
		fmt.Printf("  - %s: %v\n", ev.EmployeeNo, ev.Timestamp)
	}
}
