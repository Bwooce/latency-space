package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"latency-space/shared/celestial"
)

// Global variable to store celestial objects
var celestialObjects []celestial.CelestialObject

// Helper function to find and check if a flag is present in os.Args
func isCommandLineFlagPresent(flagName string) bool {
	flagWithDash := "-" + flagName
	flagWithDoubleDash := "--" + flagName
	
	for i, arg := range os.Args {
		// Check for both -ssl and --ssl formats
		if arg == flagWithDash || arg == flagWithDoubleDash {
			return true
		}
		
		// Check for -ssl=true format
		if strings.HasPrefix(arg, flagWithDash+"=") || strings.HasPrefix(arg, flagWithDoubleDash+"=") {
			parts := strings.Split(arg, "=")
			if len(parts) == 2 && strings.ToLower(parts[1]) == "true" {
				return true
			}
		}
		
		// Check for the next arg after -ssl 
		if (arg == flagWithDash || arg == flagWithDoubleDash) && i < len(os.Args)-1 {
			nextArg := os.Args[i+1]
			if strings.ToLower(nextArg) == "true" {
				return true
			}
		}
	}
	
	return false
}

// collectDomains generates a list of all required domain names for the latency.space DNS setup,
// based on the celestial objects.
// NOTE: All domain names MUST be lowercase for consistent DNS resolution and SSL certificate validation.
func collectDomains() []string {
	var domains []string

	// Add base domain records.
	log.Println("Adding main domain records...")
	domains = append(domains, "@")   // Root domain (latency.space)
	domains = append(domains, "www") // www subdomain

	// Add essential service subdomains if needed.
	log.Println("Checking for essential system subdomains...")
	// Status page subdomain removed - now integrated with main site

	log.Println("Processing planets and their moons...")
	for _, planet := range celestial.GetPlanets() {
		// IMPORTANT: Always enforce lowercase for all domain parts
		planetDomain := strings.ToLower(planet.Name)
		log.Printf("Adding planet: %s → %s.latency.space", planet.Name, planetDomain)
		domains = append(domains, planetDomain)

		// Add moon subdomains (e.g., phobos.mars).
		for _, moon := range celestial.GetMoons(planet.Name) {
			// Ensure both moon and planet names are lowercase for domain consistency
			// This format must match how domains are constructed in the proxy code
			moonName := strings.ToLower(moon.Name)
			moonDomain := moonName + "." + planetDomain
			log.Printf("Adding moon: %s → %s.latency.space", moon.Name, moonDomain)
			domains = append(domains, moonDomain)
		}
	}

	log.Println("Processing spacecraft...")
	for _, sc := range celestial.GetSpacecraft() {
		// Replace spaces with hyphens and ensure lowercase for domain names
		// NOTE: Hyphens are preferred over underscores for DNS compatibility
		originalName := sc.Name
		scDomain := strings.ToLower(strings.ReplaceAll(originalName, " ", "-"))
		log.Printf("Adding spacecraft: %s → %s.latency.space", originalName, scDomain)
		domains = append(domains, scDomain)
	}

	log.Println("Processing dwarf planets...")
	for _, dwarf := range celestial.GetDwarfPlanets() {
		dwarfDomain := strings.ToLower(dwarf.Name)
		log.Printf("Adding dwarf planet: %s → %s.latency.space", dwarf.Name, dwarfDomain)
		domains = append(domains, dwarfDomain)
	}

	log.Println("Processing asteroids...")
	for _, asteroid := range celestial.GetAsteroids() {
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

// ExecuteSetupDNS is the main function for DNS setup that can be called from either package
func ExecuteSetupDNS(objects []celestial.CelestialObject) {
	// Store the provided celestial objects in our local variable
	celestialObjects = objects
	
	// Define command-line flags
	var (
		apiToken = flag.String("token", "", "Cloudflare API Token (required)")
		serverIP = flag.String("ip", "", "Server IP Address to point records to (required)")
		zoneName = flag.String("zone", "latency.space", "Cloudflare Zone Name")
		autoSSL  = flag.Bool("ssl", false, "Automatically manage SSL certificates with certbot")
	)
	
	// Check if flags are already parsed (may happen if called from another package)
	if !flag.Parsed() {
		flag.Parse()
	}

	// Validate required flags
	if *apiToken == "" || *serverIP == "" {
		log.Fatal("Error: Cloudflare API Token (-token) and Server IP Address (-ip) are required.")
	}

	log.Printf("Initialized %d celestial objects for DNS setup", len(celestialObjects))

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

	log.Println("\nDNS setup completed. Starting SSL certificate management...")
	log.Println("======================================================")
	
	// Check if the --ssl flag is present directly in os.Args as a fallback
	directSSLFlag := isCommandLineFlagPresent("ssl")
	
	// Use the autoSSL flag that was defined at the top of main() or direct detection
	if *autoSSL || directSSLFlag {
		log.Printf("SSL flag detected: autoSSL=%v, directSSLFlag=%v\n", *autoSSL, directSSLFlag)
		log.Println("Automatic SSL certificate management requested")
		
		// Execute the certbot command to request/renew certificates
		// We need to ensure certbot is installed
		log.Println("Checking if certbot is installed...")
		certbotCmd := "which certbot"
		certbotOutput, certbotErr := exec.Command("bash", "-c", certbotCmd).Output()
		if certbotErr != nil || len(certbotOutput) == 0 {
			log.Println("Certbot not found, attempting to install...")
			
			// Try apt-get first, then dnf if apt-get not available
			aptCmd := "apt-get update && apt-get install -y certbot python3-certbot-nginx"
			_, aptErr := exec.Command("bash", "-c", aptCmd).Output()
			if aptErr != nil {
				// Try dnf if apt-get fails
				dnfCmd := "dnf install -y certbot python3-certbot-nginx"
				_, dnfErr := exec.Command("bash", "-c", dnfCmd).Output()
				if dnfErr != nil {
					log.Println("Failed to install certbot. Please install it manually.")
					log.Println("Then run: sudo certbot certonly --standalone -d latency.space -d *.latency.space -d *.*.latency.space")
					return
				}
			}
			log.Println("Certbot installed successfully")
		} else {
			log.Println("Certbot is already installed")
		}
		
		// Check for existing certificates and their expiration
		sslDir := "/etc/letsencrypt/live/latency.space"
		_, sslDirErr := os.Stat(sslDir)
		if os.IsNotExist(sslDirErr) {
			log.Println("No existing SSL certificates found, requesting new ones...")
			
			// Request new certificates
			// First check if Nginx is running (we'll use the nginx plugin)
			nginxCmd := "systemctl is-active nginx"
			nginxOutput, _ := exec.Command("bash", "-c", nginxCmd).Output()
			if strings.TrimSpace(string(nginxOutput)) == "active" {
				log.Println("Nginx is running, using certbot with nginx plugin...")
				
				// Check if port 80 is accessible
				portCmd := "curl -s http://localhost:80 &>/dev/null && echo 'OK' || echo 'FAIL'"
				portOutput, _ := exec.Command("bash", "-c", portCmd).Output()
				if strings.TrimSpace(string(portOutput)) == "OK" {
					// Run certbot with nginx plugin
					certbotNginxCmd := "certbot --nginx -d latency.space -d www.latency.space"
					
					// Add all important subdomains
					for _, domain := range singleLevel {
						if domain != "@" && domain != "www" {
							certbotNginxCmd += " -d " + domain + ".latency.space"
						}
					}
					
					// Add multi-level domains
					for _, domain := range multiLevel {
						certbotNginxCmd += " -d " + domain + ".latency.space"
					}
					
					// Add --non-interactive and agree-tos for automated runs
					certbotNginxCmd += " --non-interactive --agree-tos --email admin@latency.space"
					
					log.Println("Running certbot to obtain certificates...")
					log.Println(certbotNginxCmd)
					
					certbotOutput, certbotErr := exec.Command("bash", "-c", certbotNginxCmd).CombinedOutput()
					if certbotErr != nil {
						log.Printf("Error running certbot: %v\n%s", certbotErr, string(certbotOutput))
						log.Println("Failed to obtain SSL certificates automatically.")
						log.Println("Please run certbot manually to obtain certificates.")
					} else {
						log.Println("SSL certificates obtained successfully!")
						log.Println(string(certbotOutput))
					}
				} else {
					log.Println("Port 80 is not accessible. Cannot use certbot nginx plugin.")
					log.Println("Please ensure port 80 is available and run certbot manually.")
				}
			} else {
				log.Println("Nginx is not running. Using certbot standalone mode...")
				
				// Try to stop Nginx to free port 80
				stopCmd := "systemctl stop nginx"
				exec.Command("bash", "-c", stopCmd).Run()
				
				// Run certbot in standalone mode
				certbotCmd := "certbot certonly --standalone -d latency.space -d www.latency.space"
				
				// Add all important subdomains
				for _, domain := range singleLevel {
					if domain != "@" && domain != "www" {
						certbotCmd += " -d " + domain + ".latency.space"
					}
				}
				
				// Add multi-level domains
				for _, domain := range multiLevel {
					certbotCmd += " -d " + domain + ".latency.space"
				}
				
				// Add --non-interactive and agree-tos for automated runs
				certbotCmd += " --non-interactive --agree-tos --email admin@latency.space"
				
				log.Println("Running certbot to obtain certificates...")
				log.Println(certbotCmd)
				
				certbotOutput, certbotErr := exec.Command("bash", "-c", certbotCmd).CombinedOutput()
				
				// Restart Nginx
				startCmd := "systemctl start nginx"
				exec.Command("bash", "-c", startCmd).Run()
				
				if certbotErr != nil {
					log.Printf("Error running certbot: %v\n%s", certbotErr, string(certbotOutput))
					log.Println("Failed to obtain SSL certificates automatically.")
					log.Println("Please run certbot manually to obtain certificates.")
				} else {
					log.Println("SSL certificates obtained successfully!")
					log.Println(string(certbotOutput))
				}
			}
		} else {
			log.Println("Existing SSL certificates found, checking expiration...")
			
			// Check certificate expiration
			checkCmd := "openssl x509 -enddate -noout -in " + sslDir + "/fullchain.pem | cut -d= -f2"
			certDateOutput, _ := exec.Command("bash", "-c", checkCmd).Output()
			certDate := strings.TrimSpace(string(certDateOutput))
			
			// Parse the expiration date
			expiryCmd := "date -d \"" + certDate + "\" +%s"
			expirySecs, _ := exec.Command("bash", "-c", expiryCmd).Output()
			
			nowCmd := "date +%s"
			nowSecs, _ := exec.Command("bash", "-c", nowCmd).Output()
			
			// Calculate days remaining
			expiryInt, _ := strconv.ParseInt(strings.TrimSpace(string(expirySecs)), 10, 64)
			nowInt, _ := strconv.ParseInt(strings.TrimSpace(string(nowSecs)), 10, 64)
			daysRemaining := (expiryInt - nowInt) / 86400
			
			log.Printf("SSL certificates expire in %d days", daysRemaining)
			
			if daysRemaining < 30 {
				log.Println("Certificates expire in less than 30 days, attempting renewal...")
				
				renewCmd := "certbot renew --quiet"
				renewOutput, renewErr := exec.Command("bash", "-c", renewCmd).CombinedOutput()
				if renewErr != nil {
					log.Printf("Error renewing certificates: %v\n%s", renewErr, string(renewOutput))
				} else {
					log.Println("Certificate renewal completed successfully!")
				}
			} else {
				log.Println("Certificates are valid for more than 30 days, no renewal needed.")
			}
		}
	} else {
		log.Println("\nDNS setup script finished.")
		log.Println("==========================")
		log.Println("Next steps:")
		log.Println("1. Obtain SSL certificates with: sudo certbot certonly --standalone -d latency.space -d *.latency.space -d *.*.latency.space")
		log.Println("2. Configure Nginx to use the certificates")
		log.Println("3. Test all domains to ensure they resolve correctly")
		log.Println("4. To automatically manage SSL certificates, run this tool with the -ssl flag")
	}
}