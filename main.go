package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

//TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>

func main() {
	db, err := sql.Open("mysql", "root:PassW0rd@tcp(localhost:3306)/mysti")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	// See "Important settings" section.
	db.SetConnMaxLifetime(time.Minute * 3)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	row, err := db.Query("select container_id, container_name from container_stats group by container_id, container_name")
	if err != nil {
		panic(err)
	}
	defer row.Close()

	containerIds := make([]string, 30)
	for row.Next() {
		var containerId string
		var containerName string
		err = row.Scan(&containerId, &containerName)
		if err != nil {
			panic(err)
		}
		containerIds = append(containerIds, containerId)
		fmt.Println("container_id:", containerId)
	}

	// Prepare statement for reading data
	stmtOut, err := db.Prepare("SELECT __time, container_id FROM container_stats WHERE container_id = ?")
	if err != nil {
		panic(err.Error()) // proper error handling instead of panic in your app
	}
	defer stmtOut.Close()
	row, err = stmtOut.Query("79cc29057c82b4f705b7dd0022bc95dc63ea60d92e21543985d1fc4b621967d4")
	if err != nil {
		panic(err.Error())
	}

	for row.Next() {
		var timestamp string
		var containerId string
		err = row.Scan(&timestamp, &containerId)
		if err != nil {
			panic(err.Error())
		}
		fmt.Println("container_id:", containerId)
	}
}
