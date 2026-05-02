package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func main() {
	// Call the sync API
	url := "http://localhost:8080/api/devices/read-events?from=2026-04-01&to=2026-04-26"
	req, _ := http.NewRequest("POST", url, nil)
	
	// Add auth header if needed. Since I don't have a token, I'll try to find one or just assume it's running locally without strict check?
	// No, the router requires JWT.
	
	fmt.Println("Attempting to trigger sync via API...")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nBody: %s\n", resp.StatusCode, string(body))
}
