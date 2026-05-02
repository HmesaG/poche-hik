package main

import (
	"context"
	"fmt"
	"ponches/internal/config"
	"ponches/internal/hikvision"
	"ponches/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		return
	}
	repo, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		fmt.Printf("DB error: %v\n", err)
		return
	}

	values, _ := repo.GetAllConfig(context.Background())
	config.ApplyOverrides(cfg, values)

	targetDeviceIP := cfg.HikvisionIP
	client := hikvision.NewClient(targetDeviceIP, 80, cfg.HikvisionUsername, cfg.HikvisionPassword)

	payload := []byte(`{
		"searchResultPosition": 0,
		"maxResults": 1,
		"faceLibType": "blackList",
		"FDID": "1",
		"FPID": "104"
	}`)

	resp, err := client.Do(context.Background(), "POST", "/ISAPI/Intelligent/FDLib/FCSearch?format=json", nil, payload)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n", string(resp))
}
