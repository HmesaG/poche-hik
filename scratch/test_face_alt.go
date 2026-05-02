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

	endpoints := []string{
		"/ISAPI/Intelligent/FDLib/FaceDataRecord/picture?FDID=1&FPID=104",
		"/ISAPI/AccessControl/FaceDataRecord/picture?FDID=1&FPID=104",
	}

	for _, endpoint := range endpoints {
		fmt.Printf("Testing %s...\n", endpoint)
		resp, err := client.Do(context.Background(), "GET", endpoint, nil, nil)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}
		if len(resp) < 500 {
			fmt.Printf("  Response: %s\n", string(resp))
		} else {
			fmt.Printf("  Success! Received %d bytes\n", len(resp))
		}
	}
}
