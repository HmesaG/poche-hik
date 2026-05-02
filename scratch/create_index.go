package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", `C:\Users\Hector\Desktop\Grupo MV\Proyectos\Ponches\ponches.db`)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_attendance_unique ON attendance_events (device_id, employee_no, timestamp, type);`)
	if err != nil {
		fmt.Printf("Error creating index: %v\n", err)
	} else {
		fmt.Println("Index created successfully or already exists.")
	}
}
