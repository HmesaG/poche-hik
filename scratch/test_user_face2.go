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

	client := hikvision.NewClient(cfg.HikvisionIP, 80, cfg.HikvisionUsername, cfg.HikvisionPassword)

	users, err := client.GetUsers(context.Background())
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	for _, u := range users {
		if u.EmployeeNo == "104" {
			fmt.Printf("User 104: %+v\n", u)
		}
	}
}
