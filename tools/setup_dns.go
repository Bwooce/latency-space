// tools/setup_dns.go
package main

import (
	"context"
	"flag"
	"github.com/cloudflare/cloudflare-go"
	"log"
)

// Use the same config from the symlinked config.go
// The variables solarSystem and spacecraft will be available

func collectDomains() []string {
	var domains []string

	// Add the main domain records
	log.Printf("Adding main domain records...")
	domains = append(domains, "@")   // Root domain (latency.space)
	domains = append(domains, "www") // www subdomain

	// Add essential system subdomains
	log.Printf("Adding essential system subdomains...")
	domains = append(domains, "status")
	domains = append(domains, "docs")

	log.Printf("Processing solar system bodies...")

	for _, planet := range getPlanets() {
		log.Printf("Adding planet: %s", planet.Name)
		domains = append(domains, planet.Name)

		// Add moons for each planet
		for _, moon := range getMoons(planet.Name) {
			moonDomain := moon.Name + "." + planet.Name
			log.Printf("Adding moon: %s", moonDomain)
			domains = append(domains, moonDomain)
		}
	}

	log.Printf("Processing spacecraft...")
	for _, spacecraft := range getSpacecraft() {
		log.Printf("Adding spacecraft: %s", spacecraft.Name)
		domains = append(domains, spacecraft.Name)
	}

	log.Printf("Processing dwarf planets...")
	for _, dwarf := range getDwarfPlanets() {
		log.Printf("Adding spacecraft: %s", dwarf.Name)
		domains = append(domains, dwarf.Name)
	}

	log.Printf("Processing asteroids...")
	for _, asteroid := range getAsteroids() {
		log.Printf("Adding asteroid: %s", asteroid.Name)
		domains = append(domains, asteroid.Name)
	}

	return domains
}

func main() {
	var (
		apiToken = flag.String("token", "", "Cloudflare API Token")
		serverIP = flag.String("ip", "", "Server IP Address")
		zoneName = flag.String("zone", "latency.space", "Zone Name")
	)
	flag.Parse()

	if *apiToken == "" || *serverIP == "" {
		log.Fatal("Please provide -token and -ip flags")
	}

	// Get all domains from the configuration
	domains := collectDomains()
	log.Printf("Found %d domains in configuration", len(domains))
	for _, domain := range domains {
		log.Printf("Domain: %s", domain)
	}

	// Initialize Cloudflare API
	api, err := cloudflare.NewWithAPIToken(*apiToken)
	if err != nil {
		log.Fatal(err)
	}

	// Get zone ID
	zoneID, err := api.ZoneIDByName(*zoneName)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Process each domain
	for _, domain := range domains {
		log.Printf("Processing domain: %s.%s", domain, *zoneName)

		// Determine if domain should be proxied through Cloudflare
		// Only the main domain should be proxied through Cloudflare
		// All subdomains (planets, moons, spacecraft, status) must bypass Cloudflare
		// to allow the long connection timeouts needed for simulating interplanetary latency
		isProxied := false

		// Only proxy the main domain through Cloudflare
		if domain == "@" || domain == "www" {
			isProxied = true
			log.Printf("Setting up %s domain with Cloudflare proxying", domain)
		} else {
			log.Printf("Setting up %s subdomain to bypass Cloudflare (required for interplanetary latency simulation)", domain)
		}

		// Check if record exists
		records, _, err := api.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
			Type: "A",
			Name: domain + "." + *zoneName,
		})
		if err != nil {
			log.Printf("Error checking record %s: %v", domain, err)
			continue
		}

		if len(records) > 0 {
			// Update existing record
			for _, existing := range records {
				// Use the isProxied value determined above

				updateParams := cloudflare.UpdateDNSRecordParams{
					ID:      existing.ID,
					Type:    "A",
					Name:    domain,
					Content: *serverIP,
					Proxied: cloudflare.BoolPtr(isProxied),
					TTL:     1,
				}
				_, err := api.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), updateParams)
				if err != nil {
					log.Printf("Error updating record %s: %v", domain, err)
				} else {
					log.Printf("Updated record: %s.%s", domain, *zoneName)
				}
			}
		} else {
			// Create new record
			// Use the isProxied value determined above

			createParams := cloudflare.CreateDNSRecordParams{
				Type:    "A",
				Name:    domain,
				Content: *serverIP,
				Proxied: cloudflare.BoolPtr(isProxied),
				TTL:     1,
			}
			_, err := api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), createParams)
			if err != nil {
				log.Printf("Error creating record %s: %v", domain, err)
			} else {
				log.Printf("Created record: %s.%s", domain, *zoneName)
			}
		}
	}

	log.Println("DNS setup completed")
}
