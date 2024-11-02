package main

import (
	c "github.com/bariiss/echo-ip/cache"
	h "github.com/bariiss/echo-ip/handlers"
	"log"
	"net/http"
	"os"
	"time"
)

// Cleanup cache every 5 minutes to remove expired entries
func init() {
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			c.Cache.Lock()
			for ip, entry := range c.Cache.Data {
				if time.Now().After(entry.Expiration) {
					delete(c.Cache.Data, ip)
				}
			}
			c.Cache.Unlock()
		}
	}()
}

// main function initializes the server and sets up the HTTP handler
func main() {
	port := os.Getenv("ECHO_IP_PORT")
	if port == "" {
		port = "8745"
	}

	http.HandleFunc("/", h.MainHandler)
	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
