package utils

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	c "github.com/bariiss/echo-ip/cache"
	s "github.com/bariiss/echo-ip/structs"
	g "github.com/oschwald/geoip2-golang"
)

// GetClientIP retrieves the client's IP address from the request
func GetClientIP(r *http.Request) string {
	ip := r.URL.Query().Get("ip")
	if ip != "" {
		return ip
	}

	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return strings.Split(forwarded, ",")[0]
	}

	ip = r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

// FetchGeoInfo retrieves the geo information for the given IP address
// The old function signature is kept for backward compatibility, but cityDB and asnDB parameters will be ignored
func FetchGeoInfo(ip string, cityDB, asnDB *g.Reader) (*s.GeoInfo, error) {
	return FetchGeoInfoFromMemory(ip)
}

// FetchGeoInfoFromMemory retrieves the geo information for the given IP address using the in-memory DB
func FetchGeoInfoFromMemory(ip string) (*s.GeoInfo, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	// Get the database instance from our singleton
	db := GetGeoDB()

	cityRecord, err := db.CityDB.City(parsedIP)
	if err != nil {
		return nil, err
	}

	asnRecord, err := db.ASNDB.ASN(parsedIP)
	if err != nil {
		return nil, err
	}

	return &s.GeoInfo{
		ClientIP:   ip,
		Country:    cityRecord.Country.Names["en"],
		City:       cityRecord.City.Names["en"],
		Timezone:   cityRecord.Location.TimeZone,
		PostalCode: cityRecord.Postal.Code,
		Latitude:   cityRecord.Location.Latitude,
		Longitude:  cityRecord.Location.Longitude,
		ISP:        asnRecord.AutonomousSystemOrganization,
	}, nil
}

// GetGeoInfo sends a request to the server and retrieves IP info
func GetGeoInfo(ip string) (*s.GeoInfo, error) {
	url := os.Getenv("ECHO_IP_SERVICE_URL")
	if url == "" {
		url = "http://localhost:8745"
	}

	if ip != "" {
		url += "?ip=" + ip
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("error closing response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get valid response: %s", resp.Status)
	}

	var geoInfo s.GeoInfo
	if err := json.NewDecoder(resp.Body).Decode(&geoInfo); err != nil {
		return nil, fmt.Errorf("error decoding JSON response: %v", err)
	}

	return &geoInfo, nil
}

// GenerateGUID generates a random GUID
func GenerateGUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(b)
}

// CacheCleanup periodically cleans up the cache
func CacheCleanup() {
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
