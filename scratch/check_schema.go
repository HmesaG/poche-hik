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

	var schema string
	err = db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='attendance_events'").Scan(&schema)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Schema for attendance_events:\n%s\n", schema)

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='attendance_events'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	fmt.Println("Indices for attendance_events:")
	for rows.Next() {
		var name string
		rows.Scan(&name)
		fmt.Printf(" - %s\n", name)
	}
}
