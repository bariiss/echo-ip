package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	c "github.com/bariiss/echo-ip/cache"
	"github.com/bariiss/echo-ip/utils"
)

// IPMainHandler is the main HTTP handler for the server
func IPMainHandler(w http.ResponseWriter, r *http.Request) {
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

	// Query new information using the in-memory database
	geoInfo, err := utils.FetchGeoInfoFromMemory(clientIP)
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

// InitDatabases initializes the database readers (for backward compatibility)
func InitDatabases() {
	// This now just calls the singleton initializer
	utils.GetGeoDB()
}

// CloseDBs closes database readers (for backward compatibility)
func CloseDBs() {
	// Get the instance and close it
	db := utils.GetGeoDB()
	db.Close()
}
