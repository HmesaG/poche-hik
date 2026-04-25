package main

import (
	"context"
	"fmt"
	"ponches/internal/store"
)

func main() {
	s, err := store.NewSQLiteStore("ponches.db")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	events, err := s.GetEvents(context.Background(), store.EventFilter{})
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Total events in DB: %d\n", len(events))
	for i, e := range events {
		fmt.Printf("[%d] Device: %s, Emp: %s, Time: %v, Type: %s\n", i, e.DeviceID, e.EmployeeNo, e.Timestamp, e.Type)
		if i >= 10 {
			break
		}
	}
}
