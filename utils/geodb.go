package utils

import (
	"log"
	"sync"

	g "github.com/oschwald/geoip2-golang"
)

// GeoDB provides a singleton for accessing GeoIP databases
type GeoDB struct {
	CityDB *g.Reader
	ASNDB  *g.Reader
}

var (
	instance *GeoDB
	once     sync.Once
)

// GetGeoDB returns the singleton GeoDB instance
func GetGeoDB() *GeoDB {
	once.Do(func() {
		var err error
		db := &GeoDB{}

		// Open City database
		db.CityDB, err = g.Open("geolite/GeoLite2-City.mmdb")
		if err != nil {
			log.Fatal("Failed to open GeoLite2-City database:", err)
		}

		// Open ASN database
		db.ASNDB, err = g.Open("geolite/GeoLite2-ASN.mmdb")
		if err != nil {
			log.Fatal("Failed to open GeoLite2-ASN database:", err)
		}

		log.Println("GeoIP databases loaded successfully")
		instance = db
	})

	return instance
}

// Close closes all open database readers
func (db *GeoDB) Close() {
	if db.CityDB != nil {
		if err := db.CityDB.Close(); err != nil {
			log.Printf("Error closing City database: %v", err)
		}
		db.CityDB = nil
	}

	if db.ASNDB != nil {
		if err := db.ASNDB.Close(); err != nil {
			log.Printf("Error closing ASN database: %v", err)
		}
		db.ASNDB = nil
	}
}
