package main

import (
	"github.com/bariiss/echo-ip/structs"
	"sync"
	"time"
)

// Cached response structure
type cacheEntry struct {
	Response   *structs.GeoInfo
	Expiration time.Time
}

// Cache structure with mutex for concurrent access
var cache = struct {
	sync.RWMutex
	data map[string]cacheEntry
}{data: make(map[string]cacheEntry)}

// Cleanup cache every 5 minutes to remove expired entries
func init() {
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			cache.Lock()
			for ip, entry := range cache.data {
				if time.Now().After(entry.Expiration) {
					delete(cache.data, ip)
				}
			}
			cache.Unlock()
		}
	}()
}
