package main

import (
	"fmt"
	"github.com/bariiss/echo-ip/structs"
	"github.com/oschwald/geoip2-golang"
	"net"
	"net/http"
	"strings"
)

// getClientIP retrieves the client's IP address from the request
func getClientIP(r *http.Request) string {
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

// getGeoInfo retrieves the geo information for the given IP address
func getGeoInfo(ip string, cityDB, asnDB *geoip2.Reader) (*structs.GeoInfo, error) {
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
		ClientIP:  ip,
		Country:   cityRecord.Country.Names["en"],
		City:      cityRecord.City.Names["en"],
		Timezone:  cityRecord.Location.TimeZone,
		Latitude:  cityRecord.Location.Latitude,
		Longitude: cityRecord.Location.Longitude,
		ISP:       asnRecord.AutonomousSystemOrganization,
	}, nil
}
