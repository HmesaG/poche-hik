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

	rows, err := db.Query("SELECT employee_no, first_name, last_name FROM employees")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("Employees in DB:")
	for rows.Next() {
		var no, fn, ln string
		rows.Scan(&no, &fn, &ln)
		fmt.Printf(" - '%s': %s %s\n", no, fn, ln)
	}
}
