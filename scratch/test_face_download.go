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
	if targetDeviceIP == "" {
		fmt.Println("No device IP configured")
		return
	}

	client := hikvision.NewClient(targetDeviceIP, 80, cfg.HikvisionUsername, cfg.HikvisionPassword)

	endpoints := []string{
		"/ISAPI/Intelligent/FDLib/FCSearch/picture?faceLibType=blackList&FDID=1&FPID=104",
		"/ISAPI/Intelligent/FDLib/FaceDataRecord?format=json&FDID=1&FPID=104",
		"/ISAPI/AccessControl/FaceInfo/Search?employeeNo=104",
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
