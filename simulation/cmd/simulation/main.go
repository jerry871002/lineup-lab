package main

import (
	"io"
	"log"
	"net/http"
	"os"

	"github.com/jerry871002/lineup-lab/simulation/internal/api"
	"github.com/jerry871002/lineup-lab/simulation/internal/simulation"
)

func main() {
	debugLogger, infoLogger, debugMode := configureLoggers()
	simulation.ConfigureLoggers(debugLogger, infoLogger)

	port := getEnv("PORT", "80")
	allowedOrigin := getEnv("ALLOWED_ORIGIN", "*")

	handler := api.NewHandler(debugMode, allowedOrigin)

	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func configureLoggers() (*log.Logger, *log.Logger, bool) {
	debugLogger := log.New(
		os.Stdout,
		"DEBUG: ",
		log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile,
	)
	infoLogger := log.New(
		os.Stdout,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile,
	)

	debugMode := os.Getenv("DEBUG") == "1" || os.Getenv("DEBUG") == "true"
	if !debugMode {
		debugLogger.SetOutput(io.Discard)
	}

	return debugLogger, infoLogger, debugMode
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
