package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "ponches.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	columns := []string{"fleet_no", "personal_no"}
	for _, col := range columns {
		_, err := db.Exec(fmt.Sprintf("ALTER TABLE employees ADD COLUMN %s TEXT", col))
		if err != nil {
			fmt.Printf("Column %s might already exist: %v\n", col, err)
		} else {
			fmt.Printf("Added column %s\n", col)
		}
	}
	fmt.Println("Migration finished")
}
