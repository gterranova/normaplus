package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func main() {
	cwd, _ := os.Getwd()
	dbPath := filepath.Join(cwd, "backend", "normattiva.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var schema string
	err = db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name='annotations'").Scan(&schema)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Annotations Schema:")
	fmt.Println(schema)

	rows, err := db.Query("PRAGMA table_info(annotations)")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	fmt.Println("\nColumns:")
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull int
		var dflt_value interface{}
		var pk int
		rows.Scan(&cid, &name, &dtype, &notnull, &dflt_value, &pk)
		fmt.Printf("- %s (%s)\n", name, dtype)
	}
}
