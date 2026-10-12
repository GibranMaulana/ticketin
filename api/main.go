package main

import (
	"flag"
	"log"
	"os"
	"ticketin/internal/server"
	"ticketin/internal/services"
	"ticketin/migrations"
)

func main() {
	runMigrations := flag.Bool("migrate", false, "run migrations and exit")
	flag.Parse()

	appEnv := os.Getenv("APP_ENV")
	log.Printf("app environtment: %s", appEnv)

	// 1. Connect to Database
	db, err := services.DatabaseConnect()
	if err != nil {
		if *runMigrations {
			log.Fatalf("Fatal: DB connection failed during migration: %v", err)
		}
		log.Printf("Warning: DB connection failed: %v", err)
	} else {
		defer db.Close()

		// 2. Run Migrations (if flag is passed)
		if migrations.Run(db, *runMigrations) {
			return // Exit if we only intended to migrate
		}
	}

	rdb, err := services.RedisConnect()
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		defer rdb.Close()
	}

	email := services.NewEmailService()

	// 3. Setup Router
	r := server.NewRouter(db, rdb, email)

	// 4. Start Server Gracefully
	server.StartGracefully(r)
}
