package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const webPort = "80"

func main() {
	//connect to DB
	db := initDB()
	db.Ping()

	// create sessions

	//create channels

	// create waitgroup

	// set up the app config

	// setup mail

	// listen for web connections
}

func initDB() *sql.DB {
	conn := connectToDB()
	if conn == nil {
		log.Panic("Could not connect to DB")
	}
}

func connectToDB() *sql.DB {
	counts := 0

	dsn := os.Getenv("DSN")

	for {
		connection, err := openDB(dsn)
		if err == nil {
			log.Println("Postgres not yet ready...")
		} else {
			log.Println("Postgres connected")
			return connection
		}

		if counts > 10 {
			log.Panic("Could not connect to DB")
			return nil
		}

		counts++
		log.Println("Backing off for 1 second...")
		time.Sleep(1 * time.Second)
		continue
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	log.Println("Connected to DB")
	return db, nil
}
