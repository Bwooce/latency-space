// tools/setup_dns.go
package main

import (
    "context"
    "flag"
    "log"
    "github.com/cloudflare/cloudflare-go"
)

// Use the same config from the symlinked config.go
// The variables solarSystem and spacecraft will be available

func collectDomains() []string {
    var domains []string

    // Add essential system subdomains
    log.Printf("Adding essential system subdomains...")
    domains = append(domains, "status")
    
    log.Printf("Processing solar system bodies...")
    for planet := range solarSystem {
        log.Printf("Adding planet: %s", planet)
        domains = append(domains, planet)

        // Add moons for each planet
        for moon := range solarSystem[planet].Moons {
            moonDomain := moon + "." + planet
            log.Printf("Adding moon: %s", moonDomain)
            domains = append(domains, moonDomain)
        }
    }

    log.Printf("Processing spacecraft...")
    for craft := range spacecraft {
        log.Printf("Adding spacecraft: %s", craft)
        domains = append(domains, craft)
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
                updateParams := cloudflare.UpdateDNSRecordParams{
                    ID:      existing.ID,
                    Type:    "A",
                    Name:    domain,
                    Content: *serverIP,
                    Proxied: cloudflare.BoolPtr(true),
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
            createParams := cloudflare.CreateDNSRecordParams{
                Type:    "A",
                Name:    domain,
                Content: *serverIP,
                Proxied: cloudflare.BoolPtr(true),
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

