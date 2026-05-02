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

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM attendance_events").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Total events in DB: %d\n", count)

	rows, err := db.Query("SELECT * FROM attendance_events LIMIT 5")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Raw sample rows:")
	for rows.Next() {
		// Just dump whatever is there
		cols, _ := rows.Columns()
		values := make([]interface{}, len(cols))
		pointers := make([]interface{}, len(cols))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			log.Fatal(err)
		}
		for i, col := range cols {
			fmt.Printf("%s: %v | ", col, values[i])
		}
		fmt.Println()
	}
}
