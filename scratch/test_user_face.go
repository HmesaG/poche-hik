package main

import (
	"context"
	"fmt"
	"ponches/internal/config"
	"ponches/internal/hikvision"
	"ponches/internal/store"
)

func main() {
	cfg, _ := config.Load()
	repo, _ := store.NewSQLiteStore(cfg.DBPath)
	values, _ := repo.GetAllConfig(context.Background())
	config.ApplyOverrides(cfg, values)

	targetDeviceIP := cfg.HikvisionIP
	client := hikvision.NewClient(targetDeviceIP, 80, cfg.HikvisionUsername, cfg.HikvisionPassword)

	user, err := client.GetUser(context.Background(), "104")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("User: %+v\n", user)
}
