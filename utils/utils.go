package utils

import (
	"encoding/json"
	"fmt"
	"github.com/bariiss/echo-ip/structs"
	"github.com/oschwald/geoip2-golang"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
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
func FetchGeoInfo(ip string, cityDB, asnDB *geoip2.Reader) (*structs.GeoInfo, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address")
	}

	cityRecord, err := cityDB.City(parsedIP)
	if err != nil {
		return nil, err
	}

	asnRecord, err := asnDB.ASN(parsedIP)
	if err != nil {
		return nil, err
	}

	return &structs.GeoInfo{
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
func GetGeoInfo(ip string) (*structs.GeoInfo, error) {
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

	var geoInfo structs.GeoInfo
	if err := json.NewDecoder(resp.Body).Decode(&geoInfo); err != nil {
		return nil, fmt.Errorf("error decoding JSON response: %v", err)
	}

	return &geoInfo, nil
}
