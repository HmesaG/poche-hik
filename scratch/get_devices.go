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

	val, err := s.GetConfigValue(context.Background(), "managed_devices")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Println(val)
}
