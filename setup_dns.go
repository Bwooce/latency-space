// tools/setup_dns.go
package main

import (
    "context"
    "log"
    "os"
    "github.com/cloudflare/cloudflare-go"
)

func main() {
    // Get API token from environment
    apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
    zoneName := "latency.space"

    api, err := cloudflare.NewWithAPIToken(apiToken)
    if err != nil {
        log.Fatal(err)
    }

    // Get zone ID
    zoneID, err := api.ZoneIDByName(zoneName)
    if err != nil {
        log.Fatal(err)
    }

    // Define all our celestial bodies
    records := []struct {
        Name string
        Type string
    }{
        {"mars", "A"},
        {"venus", "A"},
        {"jupiter", "A"},
        {"saturn", "A"},
        {"uranus", "A"},
        {"neptune", "A"},
        {"pluto", "A"},
        // Moons
        {"phobos.mars", "A"},
        {"deimos.mars", "A"},
        {"europa.jupiter", "A"},
        {"titan.saturn", "A"},
        // Spacecraft
        {"voyager1", "A"},
        {"voyager2", "A"},
        {"jwst", "A"},
        // Wildcard
        {"*", "A"},
    }

    // Server IP
    serverIP := os.Getenv("SERVER_IP")

    // Create records
    for _, record := range records {
        _, err := api.CreateDNSRecord(context.Background(), cloudflare.ZoneIdentifier(zoneID), cloudflare.CreateDNSRecordParams{
            Type:    record.Type,
            Name:    record.Name,
            Content: serverIP,
            Proxied: cloudflare.BoolPtr(true), // Enable Cloudflare proxy
            TTL:     1, // Auto TTL
        })
        if err != nil {
            log.Printf("Error creating record %s: %v", record.Name, err)
            continue
        }
        log.Printf("Created record: %s.%s", record.Name, zoneName)
    }
}

