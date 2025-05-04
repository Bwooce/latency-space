package main

import (
	"log"
)

// setup_dns.go is the main entry point for the DNS setup tool.
// It loads celestial object data from the proxy src and calls the shared setup logic.

// Build a standalone tools/setup_dns binary that imports the object data
// from the proxy package.

func main() {
	log.Println("Starting latency.space DNS setup tool...")
	log.Println("Loading celestial object data...")
	
	// Initialize the data from objects_data.go which is imported via the package
	objects := InitSolarSystemObjects()
	log.Printf("Loaded %d celestial objects", len(objects))
	
	// Call the shared setup function with the loaded objects
	ExecuteSetupDNS(objects)
}