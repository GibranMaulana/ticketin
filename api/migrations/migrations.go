package migrations

import (
	"embed"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

// Run checks if the migrate flag is set and executes migrations if true.
// Returns true if migrations were run (so the app can exit).
func Run(db *sqlx.DB, shouldMigrate bool) bool {
	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("fatal: Set Dialect error: %v", err)
	}

	if shouldMigrate {
		log.Println("start migrations...")

		// Since this file is in the migrations folder alongside the .sql files,
		// the base path in the embedFS is just "."
		if err := goose.Up(db.DB, "."); err != nil {
			log.Fatalf("Error while migrate: %v", err)
		}

		log.Println("Migrations Completed...")
		return true
	}
	return false
}
