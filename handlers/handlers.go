package handlers

import (
	"encoding/json"
	c "github.com/bariiss/echo-ip/cache"
	"github.com/bariiss/echo-ip/utils"
	g "github.com/oschwald/geoip2-golang"
	"log"
	"net/http"
	"time"
)

// MainHandler is the main HTTP handler for the server
func MainHandler(w http.ResponseWriter, r *http.Request) {
	clientIP := utils.GetClientIP(r)

	// Check if we have a cached response
	c.Cache.RLock()
	entry, found := c.Cache.Data[clientIP]
	c.Cache.RUnlock()
	if found && time.Now().Before(entry.Expiration) {
		// Return the cached response
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(entry.Response)
		if err != nil {
			return
		}
		return
	}

	// Load databases
	city, err := g.Open("geolite/GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer func(city *g.Reader) {
		err := city.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(city)

	asn, err := g.Open("geolite/GeoLite2-ASN.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer func(asn *g.Reader) {
		err := asn.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(asn)

	// Query new information
	geoInfo, err := utils.FetchGeoInfo(clientIP, city, asn)
	if err != nil {
		http.Error(w, "Unable to retrieve geo information", http.StatusInternalServerError)
		return
	}

	// Cache the response with a 1-hour expiration
	c.Cache.Lock()
	c.Cache.Data[clientIP] = c.Entry{
		Response:   geoInfo,
		Expiration: time.Now().Add(1 * time.Hour),
	}
	c.Cache.Unlock()

	// Return the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(geoInfo); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}
