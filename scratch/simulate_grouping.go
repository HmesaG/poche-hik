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

	// Simulated 'from' and 'to' from handlers_reports.go
	// Using time.Parse like the current code (UTC)
	from, _ := time.Parse("2006-01-02", "2026-04-25")
	to, _ := time.Parse("2006-01-02", "2026-04-25")

	fmt.Printf("SIMULATION: from=%v, to=%v\n", from, to)

	// Fetch events
	allEvents, _ := sqliteStore.GetEvents(context.Background(), store.EventFilter{
		From: from,
		To:   to.Add(24 * time.Hour),
	})

	fmt.Printf("Total events fetched: %d\n", len(allEvents))

	eventsByEmp := make(map[string][]*store.AttendanceEvent)
	for _, ev := range allEvents {
		eventsByEmp[ev.EmployeeNo] = append(eventsByEmp[ev.EmployeeNo], ev)
	}

	emp104 := eventsByEmp["104"]
	fmt.Printf("Events for employee 104: %d\n", len(emp104))

	dayEventsMap := make(map[string][]*store.AttendanceEvent)
	for _, ev := range emp104 {
		dKey := ev.Timestamp.Format("2006-01-02")
		dayEventsMap[dKey] = append(dayEventsMap[dKey], ev)
		fmt.Printf("  - Mapping event %v to key %s\n", ev.Timestamp, dKey)
	}

	// Loop days
	curr := from
	for !curr.After(to) {
		dKey := curr.Format("2006-01-02")
		dayEvs := dayEventsMap[dKey]
		fmt.Printf("Day loop: key %s found %d events.\n", dKey, len(dayEvs))
		
		curr = curr.Add(24 * time.Hour)
	}
}
