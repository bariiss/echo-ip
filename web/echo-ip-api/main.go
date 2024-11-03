package main

import (
	h "github.com/bariiss/echo-ip/handlers"
	u "github.com/bariiss/echo-ip/utils"
	"log"
	"net/http"
	"os"
)

var (
	port       string
	domainName string
)

func init() {
	port = os.Getenv("ECHO_IP_PORT")         // 8745
	domainName = os.Getenv("ECHO_IP_DOMAIN") // edns.example.com.
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

	http.HandleFunc("/", h.IPMainHandler)
	http.HandleFunc("/edns", EdnsRedirectHandler)

	log.Printf("Server started on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
