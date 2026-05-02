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
		return
	}
	repo, err := store.NewSQLiteStore(cfg.DBPath)
	if err != nil {
		return
	}
	values, _ := repo.GetAllConfig(context.Background())
	config.ApplyOverrides(cfg, values)

	targetDeviceIP := cfg.HikvisionIP
	client := hikvision.NewClient(targetDeviceIP, 80, cfg.HikvisionUsername, cfg.HikvisionPassword)

	payload := []byte(`{
		"FaceInfoSearchCond": {
			"searchID": "1",
			"searchResultPosition": 0,
			"maxResults": 1,
			"EmployeeNoList": [{"employeeNo": "104"}]
		}
	}`)

	resp, err := client.Do(context.Background(), "POST", "/ISAPI/AccessControl/FaceInfo/Search?format=json", nil, payload)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Response: %s\n", string(resp))
}
