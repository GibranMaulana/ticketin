package services

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// DatabaseConnect establishes a connection to the database
func DatabaseConnect() (*sqlx.DB, error) {
	dbUrl := strings.Trim(strings.TrimSpace(os.Getenv("DB_URL")), "\"'")
	db, err := sqlx.Connect("postgres", dbUrl)
	if err != nil {
		return nil, err
	}

	// Production pooler configuration (especially for Supabase / Cloud SQL)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Connected to database!")
	return db, nil
}
