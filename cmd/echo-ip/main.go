package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/bariiss/echo-ip/structs"
	"github.com/spf13/cobra"
)

// fetchGeoInfo sends a request to the server and retrieves IP info
func fetchGeoInfo(ip string) (*structs.GeoInfo, error) {
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

// main function initializes the CLI and sets up the root command
func main() {
	var ip string

	// Define the root command
	var rootCmd = &cobra.Command{
		Use:   "echo-ip",
		Short: "Fetch IP geolocation information",
		Long:  `A CLI tool to fetch geolocation information based on an IP address using a remote server.`,
		Run: func(cmd *cobra.Command, args []string) {
			geoInfo, err := fetchGeoInfo(ip)
			if err != nil {
				log.Fatalf("Error retrieving IP information: %v", err)
			}

			fmt.Printf("Client IP: %s\n", geoInfo.ClientIP)
			fmt.Printf("Country: %s\n", geoInfo.Country)
			fmt.Printf("City: %s\n", geoInfo.City)
			fmt.Printf("Timezone: %s\n", geoInfo.Timezone)
			fmt.Printf("Latitude: %f\n", geoInfo.Latitude)
			fmt.Printf("Longitude: %f\n", geoInfo.Longitude)
			fmt.Printf("ISP: %s\n", geoInfo.ISP)
		},
	}

	// Add the "ip" flag to the root command
	rootCmd.Flags().StringVarP(&ip, "ip", "i", "", "IP address to fetch information for")

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
