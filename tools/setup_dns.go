// tools/setup_dns.go
package main

import (
	"context"
	"flag"
	"github.com/cloudflare/cloudflare-go"
	"log"
	"strings"
)

// collectDomains generates a list of all required domain names for the latency.space DNS setup,
// based on the celestial objects defined in objects_data.go.
func collectDomains() []string {
	var domains []string

	// Add base domain records.
	log.Println("Adding main domain records...")
	domains = append(domains, "@")   // Root domain (latency.space)
	domains = append(domains, "www") // www subdomain

	// Add essential service subdomains.
	log.Println("Adding essential system subdomains...")
	domains = append(domains, "status") // Status page subdomain

	log.Println("Processing planets and their moons...")
	for _, planet := range GetPlanets() {
		log.Printf("Adding planet: %s", planet.Name)
		domains = append(domains, strings.ToLower(planet.Name)) // Ensure lowercase

		// Add moon subdomains (e.g., phobos.mars).
		for _, moon := range GetMoons(planet.Name) {
			// Ensure both moon and planet names are lowercase for domain consistency
			moonDomain := strings.ToLower(moon.Name) + "." + strings.ToLower(planet.Name)
			log.Printf("Adding moon: %s", moonDomain)
			domains = append(domains, moonDomain)
		}
	}

	log.Println("Processing spacecraft...")
	for _, sc := range GetSpacecraft() {
		// Replace spaces with underscores and ensure lowercase for domain names
		scDomain := strings.ToLower(strings.ReplaceAll(sc.Name, " ", "_"))
		log.Printf("Adding spacecraft: %s", scDomain)
		domains = append(domains, scDomain)
	}

	log.Println("Processing dwarf planets...")
	for _, dwarf := range GetDwarfPlanets() {
		log.Printf("Adding dwarf planet: %s", dwarf.Name)
		domains = append(domains, strings.ToLower(dwarf.Name)) // Ensure lowercase
	}

	log.Println("Processing asteroids...")
	for _, asteroid := range GetAsteroids() {
		log.Printf("Adding asteroid: %s", asteroid.Name)
		domains = append(domains, strings.ToLower(asteroid.Name)) // Ensure lowercase
	}

	log.Printf("Collected %d domains/subdomains.", len(domains))
	return domains
}

func main() {
	// Define command-line flags
	var (
		apiToken = flag.String("token", "", "Cloudflare API Token (required)")
		serverIP = flag.String("ip", "", "Server IP Address to point records to (required)")
		zoneName = flag.String("zone", "latency.space", "Cloudflare Zone Name")
	)
	flag.Parse()

	// Validate required flags
	if *apiToken == "" || *serverIP == "" {
		log.Fatal("Error: Cloudflare API Token (-token) and Server IP Address (-ip) are required.")
	}

	// Collect all required domain names.
	domains := collectDomains()
	log.Printf("Attempting to configure DNS for %d domains/subdomains in zone '%s' pointing to IP %s",
		len(domains), *zoneName, *serverIP)

	// Initialize the Cloudflare API client.
	api, err := cloudflare.NewWithAPIToken(*apiToken)
	if err != nil {
		log.Fatalf("Error initializing Cloudflare API: %v", err)
	}

	// Get the Cloudflare Zone ID for the specified zone name.
	zoneID, err := api.ZoneIDByName(*zoneName)
	if err != nil {
		log.Fatalf("Error finding Cloudflare Zone ID for zone '%s': %v", *zoneName, err)
	}
	log.Printf("Found Zone ID '%s' for zone '%s'", zoneID, *zoneName)

	ctx := context.Background()

	// Iterate through each domain and configure its DNS record.
	for _, domain := range domains {
		fullDomainName := domain + "." + *zoneName
		if domain == "@" { // Handle root domain case
			fullDomainName = *zoneName
		}
		log.Printf("Processing DNS for: %s", fullDomainName)

		// Determine if the domain should be proxied (orange cloud) by Cloudflare.
		// Only the base domain and www should be proxied.
		// All other subdomains must bypass Cloudflare (DNS only) to allow the long
		// connection timeouts required for simulating interplanetary latency via the proxy.
		isProxied := false
		if domain == "@" || domain == "www" {
			isProxied = true
			log.Printf(" -> Setting Cloudflare proxy status: ENABLED (Orange Cloud)")
		} else {
			log.Printf(" -> Setting Cloudflare proxy status: DISABLED (DNS Only - Gray Cloud)")
		}

		// Check if an A record already exists for this domain.
		// Note: Cloudflare API expects the full domain name for listing.
		records, _, err := api.ListDNSRecords(ctx, cloudflare.ZoneIdentifier(zoneID), cloudflare.ListDNSRecordsParams{
			Type: "A",
			Name: fullDomainName,
		})
		if err != nil {
			log.Printf(" -> Error checking for existing record %s: %v", fullDomainName, err)
			continue // Skip to the next domain on error
		}

		recordParams := cloudflare.DNSRecord{
			Type:    "A",
			Name:    domain, // Use the relative name (@, www, subdomain) for create/update
			Content: *serverIP,
			Proxied: cloudflare.BoolPtr(isProxied),
			TTL:     1, // Set TTL to 1 (Automatic)
		}

		if len(records) > 0 {
			// Update the existing A record(s) - should typically only be one.
			for _, existing := range records {
				log.Printf(" -> Found existing record ID %s. Updating...", existing.ID)
				updateParams := cloudflare.UpdateDNSRecordParams{
                    ID:      existing.ID,
                    Type:    recordParams.Type,
                    Name:    recordParams.Name,
                    Content: recordParams.Content,
                    Proxied: recordParams.Proxied,
                    TTL:     recordParams.TTL,
                }

				_, err := api.UpdateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), updateParams)
				if err != nil {
					log.Printf(" -> Error updating record %s (ID: %s): %v", fullDomainName, existing.ID, err)
				} else {
					log.Printf(" -> Successfully updated record: %s", fullDomainName)
				}
			}
		} else {
			// Create a new A record.
			log.Printf(" -> No existing record found. Creating new record...")
			createParams := cloudflare.CreateDNSRecordParams{
                Type:    recordParams.Type,
                Name:    recordParams.Name,
                Content: recordParams.Content,
                Proxied: recordParams.Proxied,
                TTL:     recordParams.TTL,
            }
			_, err := api.CreateDNSRecord(ctx, cloudflare.ZoneIdentifier(zoneID), createParams)
			if err != nil {
				log.Printf(" -> Error creating record %s: %v", fullDomainName, err)
			} else {
				log.Printf(" -> Successfully created record: %s", fullDomainName)
			}
		}
	}

	log.Println("\nDNS setup script finished.")
}
