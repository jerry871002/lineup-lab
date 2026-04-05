package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/jerry871002/lineup-lab/stat-api-server/internal/api"
	"github.com/jerry871002/lineup-lab/stat-api-server/internal/store"
	_ "github.com/lib/pq"
)

func main() {
	connStr := getEnv("DATABASE_URL", "postgres://myuser:mypassword@db/mydatabase?sslmode=disable")
	port := getEnv("PORT", "80")
	allowedOrigin := getEnv("ALLOWED_ORIGIN", "*")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Println("Connected to database")

	server := api.NewServer(store.NewSQLStatStore(db))
	handler := api.NewHandler(server, allowedOrigin)

	log.Printf("Server started at :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
