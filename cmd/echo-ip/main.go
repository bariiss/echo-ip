package main

import (
	"fmt"
	"log"
	"os"

	u "github.com/bariiss/echo-ip/utils"
	"github.com/spf13/cobra"
)

// main function initializes the CLI and sets up the root command
func main() {
	var ip string

	// Define the root command
	var rootCmd = &cobra.Command{
		Use:   "echo-ip",
		Short: "Fetch IP geolocation information",
		Long:  `A CLI tool to fetch geolocation information based on an IP address using a remote server.`,
		Run: func(cmd *cobra.Command, args []string) {
			geoInfo, err := u.GetGeoInfo(ip)
			if err != nil {
				log.Fatalf("Error retrieving IP information: %v", err)
			}

			fmt.Printf("Client IP: %s\n", geoInfo.ClientIP)
			fmt.Printf("Country: %s\n", geoInfo.Country)
			fmt.Printf("City: %s\n", geoInfo.City)
			fmt.Printf("Timezone: %s\n", geoInfo.Timezone)
			fmt.Printf("Postal Code: %s\n", geoInfo.PostalCode)
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
