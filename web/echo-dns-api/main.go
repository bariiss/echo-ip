package main

import (
	"fmt"
	u "github.com/bariiss/echo-ip/utils"
	"github.com/miekg/dns"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	targetIP   string
	port       string
	domainName string
	wildcard   string
)

// ClientDNSInfo stores DNS server information for a client
type ClientDNSInfo struct {
	PrimaryDNS   string
	SecondaryDNS string
	LastSeen     time.Time
}

// DNSCache stores client DNS information with thread-safe access
type DNSCache struct {
	sync.RWMutex
	clients map[string]*ClientDNSInfo
}

// Global cache instance
var dnsCache = &DNSCache{
	clients: make(map[string]*ClientDNSInfo),
}

// Initialize the target IP and domain name
func init() {
	targetIP = os.Getenv("ECHO_IP_TARGET_IP")
	port = os.Getenv("ECHO_IP_PORT")
	domainName = os.Getenv("ECHO_IP_DOMAIN")
	wildcard = "*." + domainName

	// Start cache cleanup routine
	go cleanupCache()
}

// UpdateClientDNS updates the DNS servers for a client
func (c *DNSCache) UpdateClientDNS(clientIP, dnsIP string) {
	c.Lock()
	defer c.Unlock()

	info, exists := c.clients[clientIP]
	if !exists {
		info = &ClientDNSInfo{
			PrimaryDNS: dnsIP,
			LastSeen:   time.Now(),
		}
		c.clients[clientIP] = info
	} else if info.PrimaryDNS != dnsIP {
		if info.SecondaryDNS == "" || info.SecondaryDNS != dnsIP {
			info.SecondaryDNS = dnsIP
		}
		info.LastSeen = time.Now()
	}
}

// GetClientDNSInfo retrieves DNS information for a client
func (c *DNSCache) GetClientDNSInfo(clientIP string) *ClientDNSInfo {
	c.RLock()
	defer c.RUnlock()
	return c.clients[clientIP]
}

// cleanupCache removes old entries from the cache
func cleanupCache() {
	ticker := time.NewTicker(10 * time.Minute)
	for range ticker.C {
		dnsCache.Lock()
		now := time.Now()
		for ip, info := range dnsCache.clients {
			if now.Sub(info.LastSeen) > 24*time.Hour {
				delete(dnsCache.clients, ip)
			}
		}
		dnsCache.Unlock()
	}
}

// DNSRequestHandler handles DNS requests for GUID subdomains
func DNSRequestHandler(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	// Log request and target IP
	log.Printf("Handling DNS request. Target IP: %s, Domain: %s", targetIP, domainName)

	// Extract resolver and client IPs
	resolverIP, _, _ := net.SplitHostPort(w.RemoteAddr().String())
	clientIP := resolverIP
	for _, extra := range r.Extra {
		if opt, ok := extra.(*dns.OPT); ok {
			for _, o := range opt.Option {
				if e, ok := o.(*dns.EDNS0_SUBNET); ok {
					clientIP = e.Address.String()
					break
				}
			}
		}
	}

	// Update the DNS cache
	dnsCache.UpdateClientDNS(clientIP, resolverIP)

	// Loop through questions and prepare an A record if applicable
	for _, question := range r.Question {
		log.Printf("Question received: %v", question.Name)
		if question.Qtype == dns.TypeA && (question.Name == domainName || dns.IsSubDomain(wildcard, question.Name)) {
			ip := net.ParseIP(targetIP)
			if ip == nil {
				log.Printf("Error: Unable to parse target IP %s", targetIP)
				return
			}

			// Create the A record
			aRecord := &dns.A{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    60,
				},
				A: ip,
			}
			m.Answer = append(m.Answer, aRecord)
			log.Printf("Added A record for %s with IP %s", question.Name, targetIP)
		}
	}

	// Log the entire answer section for verification
	log.Printf("Response Answer Section: %v", m.Answer)

	// Write the response
	if err := w.WriteMsg(m); err != nil {
		log.Printf("Error writing DNS response: %v", err)
	}
}

// DNSMainHandler handles the root path and generates a new GUID
func DNSMainHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		// Get client IP
		clientIP := u.GetClientIP(r)

		// Get DNS information
		dnsInfo := dnsCache.GetClientDNSInfo(clientIP)

		if dnsInfo != nil {
			// Display DNS information and generate new GUID
			w.Header().Set("Content-Type", "text/plain")
			response := fmt.Sprintf("Client IP: %s\nPrimary DNS: %s\nSecondary DNS: %s\n\n",
				clientIP,
				dnsInfo.PrimaryDNS,
				dnsInfo.SecondaryDNS)

			guid := u.GenerateGUID()
			redirectURL := fmt.Sprintf("https://%s.%s", guid, domainName)
			response += fmt.Sprintf("Generated GUID: %s\nRedirect URL: %s", guid, redirectURL)

			_, err := fmt.Fprint(w, response)
			if err != nil {
				return
			}
		} else {
			// If no DNS info is available, just generate GUID and redirect
			guid := u.GenerateGUID()
			redirectURL := fmt.Sprintf("https://%s.%s", guid, domainName)
			http.Redirect(w, r, redirectURL, http.StatusFound)
		}
		return
	}
	GUIDRequestHandler(w, r)
}

// GUIDRequestHandler handles HTTP requests to /{guid}
func GUIDRequestHandler(w http.ResponseWriter, r *http.Request) {
	guid := strings.TrimPrefix(r.URL.Path, "/")
	if guid == "" {
		http.Error(w, "GUID is missing in the URL", http.StatusBadRequest)
		return
	}

	clientIP := u.GetClientIP(r)
	dnsInfo := dnsCache.GetClientDNSInfo(clientIP)

	w.Header().Set("Content-Type", "text/plain")

	response := fmt.Sprintf("GUID: %s\n", guid)
	if dnsInfo != nil {
		response += fmt.Sprintf("Client IP: %s\nPrimary DNS: %s\nSecondary DNS: %s\n",
			clientIP,
			dnsInfo.PrimaryDNS,
			dnsInfo.SecondaryDNS)
	}

	redirectURL := fmt.Sprintf("https://%s.%s", guid, domainName)
	response += fmt.Sprintf("Redirect URL: %s", redirectURL)

	_, err := fmt.Fprint(w, response)
	if err != nil {
		return
	}
}

func main() {
	// Initialize the DNS server
	dns.HandleFunc(".", DNSRequestHandler)
	server := &dns.Server{Addr: ":53", Net: "udp"}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set up the DNS server: %v", err)
		}
	}()
	defer func(server *dns.Server) {
		err := server.Shutdown()
		if err != nil {
			log.Fatalf("Failed to shut down the DNS server: %v", err)
		}
	}(server)

	// Initialize the HTTP server
	http.HandleFunc("/", DNSMainHandler)
	log.Printf("HTTP Server started on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
