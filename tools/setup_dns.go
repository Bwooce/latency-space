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
// NOTE: All domain names MUST be lowercase for consistent DNS resolution and SSL certificate validation.
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
		// IMPORTANT: Always enforce lowercase for all domain parts
		planetDomain := strings.ToLower(planet.Name)
		log.Printf("Adding planet: %s → %s.latency.space", planet.Name, planetDomain)
		domains = append(domains, planetDomain)

		// Add moon subdomains (e.g., phobos.mars).
		for _, moon := range GetMoons(planet.Name) {
			// Ensure both moon and planet names are lowercase for domain consistency
			// This format must match how domains are constructed in the proxy code
			moonName := strings.ToLower(moon.Name)
			moonDomain := moonName + "." + planetDomain
			log.Printf("Adding moon: %s → %s.latency.space", moon.Name, moonDomain)
			domains = append(domains, moonDomain)
		}
	}

	log.Println("Processing spacecraft...")
	for _, sc := range GetSpacecraft() {
		// Replace spaces with hyphens and ensure lowercase for domain names
		// NOTE: Hyphens are preferred over underscores for DNS compatibility
		originalName := sc.Name
		scDomain := strings.ToLower(strings.ReplaceAll(originalName, " ", "-"))
		log.Printf("Adding spacecraft: %s → %s.latency.space", originalName, scDomain)
		domains = append(domains, scDomain)
	}

	log.Println("Processing dwarf planets...")
	for _, dwarf := range GetDwarfPlanets() {
		dwarfDomain := strings.ToLower(dwarf.Name)
		log.Printf("Adding dwarf planet: %s → %s.latency.space", dwarf.Name, dwarfDomain)
		domains = append(domains, dwarfDomain)
	}

	log.Println("Processing asteroids...")
	for _, asteroid := range GetAsteroids() {
		asteroidDomain := strings.ToLower(asteroid.Name)
		log.Printf("Adding asteroid: %s → %s.latency.space", asteroid.Name, asteroidDomain)
		domains = append(domains, asteroidDomain)
	}

	// Validate all domains are lowercase (critical for SSL and DNS consistency)
	log.Println("Validating domain names...")
	for i, domain := range domains {
		if domain != "@" && domain != "www" && domain != strings.ToLower(domain) {
			log.Printf("WARNING: Domain %s contains uppercase characters. Forcing lowercase.", domain)
			domains[i] = strings.ToLower(domain)
		}
		
		// Check for problematic characters
		if strings.Contains(domain, "_") {
			log.Printf("WARNING: Domain %s contains underscores which may cause DNS issues. Consider using hyphens instead.", domain)
		}
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
		
	// Final validation of all domain names for SSL certificate compatibility
	log.Println("\nPERFORMING FINAL DOMAIN VALIDATION FOR SSL COMPATIBILITY")
	log.Println("========================================================")
	log.Println("These domains will need to be covered by SSL certificates:")
	
	// Group domains by levels for certificate planning
	var rootDomains, singleLevel, multiLevel []string
	
	for _, domain := range domains {
		// Skip special cases
		if domain == "@" || domain == "www" {
			rootDomains = append(rootDomains, domain)
			continue
		}
		
		// Check domain format and validate
		if strings.Contains(domain, ".") {
			multiLevel = append(multiLevel, domain)
		} else {
			singleLevel = append(singleLevel, domain)
		}
		
		// Ensure lowercase
		if domain != strings.ToLower(domain) {
			log.Printf("ERROR: Domain '%s' contains uppercase letters - this will cause SSL certificate validation failures!", domain)
		}
	}
	
	log.Printf("Root domains: %d", len(rootDomains))
	log.Printf("Single-level subdomains: %d (covered by *.latency.space certificate)", len(singleLevel))
	log.Printf("Multi-level subdomains: %d (require *.*.latency.space certificate)", len(multiLevel))
	
	// Suggest the certbot command
	log.Println("\nTo create SSL certificates for all these domains, use:")
	log.Printf("certbot certonly --standalone -d latency.space -d *.latency.space -d *.*.latency.space")
	
	// Initialize the Cloudflare API client.
	log.Println("\nINITIALIZING DNS CONFIGURATION")
	log.Println("=============================")
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
	
	// Counter for tracking progress
	processedCount := 0
	totalCount := len(domains)

	// Iterate through each domain and configure its DNS record.
	for _, domain := range domains {
		processedCount++
		fullDomainName := domain + "." + *zoneName
		if domain == "@" { // Handle root domain case
			fullDomainName = *zoneName
		}
		log.Printf("[%d/%d] Processing DNS for: %s", processedCount, totalCount, fullDomainName)

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
	log.Println("==========================")
	log.Println("Next steps:")
	log.Println("1. Obtain SSL certificates with: sudo certbot certonly --standalone -d latency.space -d *.latency.space -d *.*.latency.space")
	log.Println("2. Configure Nginx to use the certificates")
	log.Println("3. Test all domains to ensure they resolve correctly")
}