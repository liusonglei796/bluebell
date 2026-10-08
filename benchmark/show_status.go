package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "root:15939087780Ll@@tcp(127.0.0.1:3307)/bluebell?charset=utf8mb4"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	queries := []string{
		"DESC post_author;",
		"DESC post;",
	}

	for _, q := range queries {
		fmt.Printf("=== %s ===\n", q)
		rows, err := db.Query(q)
		if err != nil {
			fmt.Printf("执行失败: %v\n", err)
			continue
		}
		for rows.Next() {
			var name, val string
			if err := rows.Scan(&name, &val); err == nil {
				fmt.Printf("%-25s : %s\n", name, val)
			}
		}
		rows.Close()
		fmt.Println()
	}
}
