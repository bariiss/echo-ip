package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	h "github.com/bariiss/echo-ip/handlers"
	u "github.com/bariiss/echo-ip/utils"
)

var (
	port       string
	domainName string
)

func init() {
	port = os.Getenv("ECHO_IP_PORT")         // 8745
	domainName = os.Getenv("ECHO_IP_DOMAIN") // edns.example.com.

	// Start the cache cleanup routine
	u.CacheCleanup()
}

func EdnsRedirectHandler(w http.ResponseWriter, r *http.Request) {
	guid := u.GenerateGUID()
	redirectURL := "https://" + guid + "." + domainName
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func main() {
	if port == "" {
		port = "8745"
	}

	// Pre-initialize databases to avoid delay on first request
	h.InitDatabases()
	defer h.CloseDBs()

	// Create server
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.IPMainHandler)
	mux.HandleFunc("/edns", EdnsRedirectHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		<-sigChan
		log.Println("Shutting down server...")

		// Create a deadline for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Server started on :%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server successfully stopped")
}
