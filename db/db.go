package db

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type BotDB struct {
	db     *sql.DB
	locker *sync.Mutex
}

// Open DB connection
func (db *BotDB) Open(dbPath string) {
	var err error
	db.db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}

	db.locker = &sync.Mutex{}
}

func (db *BotDB) Close() {
	db.db.Close()
}

func (db *BotDB) QueryRow(query string, params ...interface{}) *sql.Row {
	db.locker.Lock()
	defer db.locker.Unlock()

	stmt, err := db.db.Prepare(query)
	if err != nil {
		fmt.Println("Crashed while querying")
		log.Fatal(err)
	}
	defer stmt.Close()

	row := stmt.QueryRow(params...)
	return row
}

func (db *BotDB) Update(query string, params ...interface{}) {
	db.locker.Lock()
	defer db.locker.Unlock()

	stmt, err := db.db.Prepare(query)
	if err != nil {
		fmt.Println("Crashed while preparing")
		log.Fatal(err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(params...)
	if err != nil {
		fmt.Println("Crashed while updating")
		log.Fatal(err)
	}
}
