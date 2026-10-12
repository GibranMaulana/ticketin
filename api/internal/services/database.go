package services

import (
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Connect establishes a connection to the database
func DatabaseConnect() (*sqlx.DB, error) {
	dbUrl := os.Getenv("DB_URL")
	db, err := sqlx.Connect("postgres", dbUrl)
	if err != nil {
		return nil, err
	}

	log.Println("Connected to database!")
	return db, nil
}
