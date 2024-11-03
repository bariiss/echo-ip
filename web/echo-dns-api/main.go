package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/miekg/dns"
)

// DNSRequest stores details of captured DNS requests.
type DNSRequest struct {
	GUID        string
	ResolverIP  string
	QueryDomain string
}

// DNSHandler manages DNS requests and stores them in memory.
type DNSHandler struct {
	requests []DNSRequest
	mu       sync.Mutex
}

// Generate a unique GUID for each request
func generateGUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(b)
}

// Handle the initial HTTP request, redirecting to the GUID-based subdomain
func handleInitialRequest(w http.ResponseWriter, r *http.Request) {
	// Get the EDNS domain from environment variable
	domain := os.Getenv("EDNS_DOMAIN")
	if domain == "" {
		log.Fatal("EDNS_DOMAIN environment variable is not set")
	}

	guid := generateGUID()
	redirectURL := fmt.Sprintf("https://%s.%s/json", guid, domain)

	// Log or save GUID to track associated requests (optional for this example)

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// ServeDNS handles DNS queries and logs resolver IP and GUID
func (h *DNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(r.Question) > 0 {
		question := r.Question[0]
		resolverIP := w.RemoteAddr().String()

		// Extract GUID from the subdomain (first segment of question.Name)
		var guid string
		if len(question.Name) > 0 {
			parts := dns.SplitDomainName(question.Name)
			if len(parts) > 0 {
				guid = parts[0] // GUID is the first part of the domain
			}
		}

		// Store the DNS request information
		dnsRequest := DNSRequest{
			GUID:        guid,
			ResolverIP:  resolverIP,
			QueryDomain: question.Name,
		}
		h.requests = append(h.requests, dnsRequest)
		log.Printf("Captured DNS request: %+v", dnsRequest)
	}

	// Respond to avoid timeout
	m := new(dns.Msg)
	m.SetReply(r)
	err := w.WriteMsg(m)
	if err != nil {
		return
	}
}

// ServeCapturedRequests provides a web endpoint to view captured DNS requests
func (h *DNSHandler) ServeCapturedRequests(w http.ResponseWriter, _ *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, req := range h.requests {
		_, _ = fmt.Fprintf(w, "GUID: %s, Resolver IP: %s, Query Domain: %s\n", req.GUID, req.ResolverIP, req.QueryDomain)
	}
}

func main() {
	dnsHandler := &DNSHandler{}

	// Start DNS server on a specified port
	go func() {
		dnsServer := &dns.Server{Addr: ":5353", Net: "udp"}
		dns.HandleFunc(".", dnsHandler.ServeDNS)
		log.Println("Starting DNS server on :5353")
		if err := dnsServer.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start DNS server: %v", err)
		}
	}()

	// HTTP server to redirect to unique GUID-based subdomain
	http.HandleFunc("/json", handleInitialRequest)
	// Endpoint to view captured DNS requests
	http.HandleFunc("/dns-requests", dnsHandler.ServeCapturedRequests)

	log.Println("Starting web server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
