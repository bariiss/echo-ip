package main

import (
	"encoding/json"
	"github.com/oschwald/geoip2-golang"
	"log"
	"net/http"
	"time"
)

// handler is the main HTTP handler for the server
func handler(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)

	// Check if we have a cached response
	cache.RLock()
	entry, found := cache.data[clientIP]
	cache.RUnlock()
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
	cityDB, err := geoip2.Open("geolite/GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer func(cityDB *geoip2.Reader) {
		err := cityDB.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(cityDB)

	asnDB, err := geoip2.Open("geolite/GeoLite2-ASN.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer func(asnDB *geoip2.Reader) {
		err := asnDB.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(asnDB)

	// Query new information
	geoInfo, err := getGeoInfo(clientIP, cityDB, asnDB)
	if err != nil {
		http.Error(w, "Unable to retrieve geo information", http.StatusInternalServerError)
		return
	}

	// Cache the response with a 1-hour expiration
	cache.Lock()
	cache.data[clientIP] = cacheEntry{
		Response:   geoInfo,
		Expiration: time.Now().Add(1 * time.Hour),
	}
	cache.Unlock()

	// Return the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(geoInfo); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}
