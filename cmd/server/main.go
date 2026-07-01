package main

import (
	"log"
	"os"

	"github.com/readflow/readflow/internal/server"
)

func main() {
	dbPath := os.Getenv("READFLOW_DB_PATH")
	if dbPath == "" {
		dbPath = "data/readflow.db"
	}

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	srv, err := server.New(dbPath)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Readflow starting on http://localhost:%s", port)
	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
