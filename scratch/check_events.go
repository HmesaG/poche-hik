package main

import (
	"context"
	"fmt"
	"ponches/internal/hikvision"
    "encoding/json"
)

func main() {
	ip := "10.0.0.100"
	user := "admin"
	pass := "Ianmesa290184@."

	client := hikvision.NewClient(ip, 80, user, pass)
	ctx := context.Background()

	start := "2026-04-24T00:00:00+08:00"
	end := "2026-04-25T23:59:59+08:00"

	reqBody := map[string]interface{}{
		"AcsEventCond": map[string]interface{}{
			"searchID":             "search-1234567890",
			"searchResultPosition": 0,
			"maxResults":           100,
			"major":                0,
			"minor":                0,
			"startTime":           start,
			"endTime":             end,
		},
	}

	body, _ := json.Marshal(reqBody)
	fmt.Printf("Sending JSON with +08:00: %s\n", string(body))

	resp, err := client.Do(ctx, "POST", "/ISAPI/AccessControl/AcsEvent?format=json", 
		map[string]string{"Content-Type": "application/json"}, body)
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
        fmt.Printf("Response: %s\n", string(resp))
		return
	}

	fmt.Printf("Success! Response: %s\n", string(resp))
}
