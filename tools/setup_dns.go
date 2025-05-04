package main

import (
	"log"
	
	"latency-space/shared/celestial"
)

// setup_dns.go is the main entry point for the DNS setup tool.
// It loads celestial object data from the shared celestial package and calls the shared setup logic.

func main() {
	log.Println("Starting latency.space DNS setup tool...")
	log.Println("Loading celestial object data from shared package...")
	
	// Initialize the data from the shared celestial package
	objects := celestial.InitSolarSystemObjects()
	log.Printf("Loaded %d celestial objects", len(objects))
	
	// Call the shared setup function with the loaded objects
	ExecuteSetupDNS(objects)
}