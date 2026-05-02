package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"database/sql"

	_ "modernc.org/sqlite"
)

type ManagedDevice struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Port      int    `json:"port"`
}

func main() {
	db, err := sql.Open("sqlite", `C:\Users\Hector\Desktop\Grupo MV\Proyectos\Ponches\ponches.db`)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var raw string
	err = db.QueryRow("SELECT value FROM app_config WHERE key = 'managed_devices'").Scan(&raw)
	if err != nil {
		log.Fatal("No managed devices found in DB (app_config):", err)
	}

	var devices []ManagedDevice
	if err := json.Unmarshal([]byte(raw), &devices); err != nil {
		log.Fatal(err)
	}

	if len(devices) == 0 {
		fmt.Println("No devices configured.")
		return
	}

	for _, d := range devices {
		fmt.Printf("Device: %s (%s)\n", d.Name, d.IP)
		checkDevice(d)
	}
}

func checkDevice(d ManagedDevice) {
	url := fmt.Sprintf("http://%s:%d/ISAPI/System/deviceInfo", d.IP, d.Port)
	req, _ := http.NewRequest("GET", url, nil)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  [ERROR] Connection failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("  [INFO] HTTP Status: %d\n", resp.StatusCode)
	
	if resp.StatusCode == 401 {
		fmt.Println("  [INFO] Device reachable but needs authentication.")
	}
}
