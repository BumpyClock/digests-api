package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	InitializeRoutes(mux) // Set up routes

	log.Println("Server is starting on port 8080...")

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
