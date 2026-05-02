package main

import (
	"context"
	"fmt"
	"ponches/internal/config"
	"ponches/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Config err: %v\n", err)
		return
	}
	repo, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		fmt.Printf("DB load err: %v\n", err)
		return
	}
	u, err := repo.GetUserByUsername(context.Background(), "admin")
	if err != nil {
		fmt.Printf("Error GetUserByUsername: %v\n", err)
		return
	}
	if u == nil {
		fmt.Printf("User not found\n")
		return
	}
	fmt.Printf("Success! User ID: %s, Role: %s\n", u.ID, u.Role)
}
