package main

import (
	"log"
	"net/http"
	"os"
)

// main function initializes the server and sets up the HTTP handler
func main() {
	// Get the port from environment variable, default to 8745 if not set
	port := os.Getenv("ECHO_IP_PORT")
	if port == "" {
		port = "8745"
	}

	http.HandleFunc("/", handler)
	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
