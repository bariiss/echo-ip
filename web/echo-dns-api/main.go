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
	targetIP  string
	port      string
	dnsDomain string // llll.ist
	nsDomain1 string // ns1.il1.nl
	nsDomain2 string // ns2.il1.nl
	wildcard  string
)

type ClientDNSInfo struct {
	PrimaryDNS   string
	SecondaryDNS string
	LastSeen     time.Time
}

type DNSCache struct {
	sync.RWMutex
	clients map[string]*ClientDNSInfo
}

var dnsCache = &DNSCache{
	clients: make(map[string]*ClientDNSInfo),
}

func init() {
	targetIP = os.Getenv("ECHO_IP_TARGET_IP")
	port = os.Getenv("ECHO_IP_PORT")
	dnsDomain = os.Getenv("ECHO_IP_DNS_DOMAIN")
	nsDomain1 = os.Getenv("ECHO_IP_NS_DOMAIN_1")
	nsDomain2 = os.Getenv("ECHO_IP_NS_DOMAIN_2")

	if !strings.HasSuffix(dnsDomain, ".") {
		dnsDomain = dnsDomain + "."
	}

	wildcard = "*." + dnsDomain

	// Validate required environment variables
	if targetIP == "" || port == "" || dnsDomain == "" || nsDomain1 == "" || nsDomain2 == "" {
		log.Fatal("Required environment variables are not set")
	}

	// Add trailing dots to NS domains if missing
	if !strings.HasSuffix(nsDomain1, ".") {
		nsDomain1 = nsDomain1 + "."
	}
	if !strings.HasSuffix(nsDomain2, ".") {
		nsDomain2 = nsDomain2 + "."
	}

	log.Printf("Initialized with Target IP: %s, DNS Domain: %s, NS1: %s, NS2: %s",
		targetIP, dnsDomain, nsDomain1, nsDomain2)

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

func isMatchingDomain(name string) bool {
	// Normalize input domain
	if !strings.HasSuffix(name, ".") {
		name = name + "."
	}

	// Root domain match
	if name == dnsDomain {
		log.Printf("Root domain match: %s", name)
		return true
	}

	// Check for edns subdomain
	ednsPrefix := "edns." + dnsDomain
	if name == ednsPrefix || strings.HasSuffix(name, "."+ednsPrefix) {
		withoutSuffix := strings.TrimSuffix(name, "."+ednsPrefix)
		// Allow exact match or direct subdomains only for edns
		isValid := name == ednsPrefix || !strings.Contains(withoutSuffix, ".")
		log.Printf("EDNS domain check: %s, valid: %v", name, isValid)
		return isValid
	}

	log.Printf("No match for domain: %s", name)
	return false
}

func DNSRequestHandler(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	log.Printf("Received DNS request from %s", w.RemoteAddr().String())

	for _, question := range r.Question {
		name := question.Name
		log.Printf("Question: %s, Type: %d", name, question.Qtype)

		switch question.Qtype {
		case dns.TypeSOA:
			if name == dnsDomain {
				soa := &dns.SOA{
					Hdr: dns.RR_Header{
						Name:   dnsDomain,
						Rrtype: dns.TypeSOA,
						Class:  dns.ClassINET,
						Ttl:    3600,
					},
					Ns:      nsDomain1, // Primary NS
					Mbox:    nsDomain2, // İkinci NS'i mail server yerine kullan
					Serial:  uint32(time.Now().Unix()),
					Refresh: 3600,
					Retry:   900,
					Expire:  86400,
					Minttl:  60,
				}
				m.Answer = append(m.Answer, soa)
				log.Printf("Added SOA record for %s", name)
			}

		case dns.TypeNS:
			if name == dnsDomain {
				ns1 := &dns.NS{
					Hdr: dns.RR_Header{
						Name:   dnsDomain,
						Rrtype: dns.TypeNS,
						Class:  dns.ClassINET,
						Ttl:    3600,
					},
					Ns: nsDomain1,
				}
				ns2 := &dns.NS{
					Hdr: dns.RR_Header{
						Name:   dnsDomain,
						Rrtype: dns.TypeNS,
						Class:  dns.ClassINET,
						Ttl:    3600,
					},
					Ns: nsDomain2,
				}
				m.Answer = append(m.Answer, ns1, ns2)
				log.Printf("Added NS records for %s", name)
			}

		case dns.TypeA:
			if isMatchingDomain(name) {
				ip := net.ParseIP(targetIP)
				if ip == nil {
					log.Printf("Error: Invalid target IP address: %s", targetIP)
					continue
				}

				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					A: ip,
				}
				m.Answer = append(m.Answer, rr)
				log.Printf("Added A record: %s -> %s", name, targetIP)
			}
		}
	}

	// Extract client information
	resolverIP, _, _ := net.SplitHostPort(w.RemoteAddr().String())
	clientIP := resolverIP

	// Update cache
	dnsCache.UpdateClientDNS(clientIP, resolverIP)

	// Log response
	log.Printf("Sending response with %d answers", len(m.Answer))
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
			redirectURL := fmt.Sprintf("https://%s.edns.%s", guid, dnsDomain)
			response += fmt.Sprintf("Generated GUID: %s\nRedirect URL: %s", guid, redirectURL)

			_, err := fmt.Fprint(w, response)
			if err != nil {
				return
			}
		} else {
			// If no DNS info is available, just generate GUID and redirect
			guid := u.GenerateGUID()
			redirectURL := fmt.Sprintf("https://%s.edns.%s", guid, dnsDomain)
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

	redirectURL := fmt.Sprintf("https://%s.edns.%s", guid, dnsDomain)
	response += fmt.Sprintf("Redirect URL: %s", redirectURL)

	_, err := fmt.Fprint(w, response)
	if err != nil {
		return
	}
}

func main() {
	// DNS server
	dns.HandleFunc(".", DNSRequestHandler) // Tüm domainleri yakala
	server := &dns.Server{Addr: ":53", Net: "udp"}

	log.Printf("Starting DNS server for domain: %s (wildcard: %s)", dnsDomain, wildcard)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start DNS server: %v", err)
		}
	}()

	// HTTP server
	http.HandleFunc("/", DNSMainHandler)
	log.Printf("Starting HTTP Server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
