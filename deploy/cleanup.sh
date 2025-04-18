#!/bin/bash
# Script to clean up temporary fix scripts once everything is working

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }

blue "Starting cleanup process..."

# Check if main site and subdomains are working
TESTS_PASSED=true

# Test main site
blue "Testing main site..."
if curl -s http://latency.space | grep -q "Latency Space"; then
    green "✓ Main site (latency.space) is working!"
else
    red "✗ Main site (latency.space) is not working."
    TESTS_PASSED=false
fi

# Test Mars subdomain
blue "Testing Mars subdomain..."
if curl -s http://mars.latency.space | grep -q ""; then
    green "✓ Mars subdomain is working!"
else
    red "✗ Mars subdomain is not working."
    TESTS_PASSED=false
fi

# Test Status subdomain
blue "Testing Status subdomain..."
if curl -s -I http://status.latency.space | grep -q "200 OK"; then
    green "✓ Status subdomain is working!"
else
    red "✗ Status subdomain is not working."
    TESTS_PASSED=false
fi

# Only proceed with cleanup if all tests pass
if [ "$TESTS_PASSED" = true ]; then
    green "All tests passed! Proceeding with cleanup."
    
    # List of temporary fix scripts to remove
    FIX_SCRIPTS=(
        "diagnostic.sh"
        "server-check.sh"
        "nginx-config-fix.sh"
        "filesystem-fix.sh"
        "check-containers.sh"
        "final-fix.sh"
        "complete-fix.sh"
    )
    
    # Create a backup directory for the scripts
    BACKUP_DIR="/opt/latency-space/deploy/old-scripts-backup"
    mkdir -p $BACKUP_DIR
    
    # Move scripts to backup directory
    for script in "${FIX_SCRIPTS[@]}"; do
        if [ -f "/opt/latency-space/deploy/$script" ]; then
            blue "Moving $script to backup directory..."
            mv "/opt/latency-space/deploy/$script" "$BACKUP_DIR/$script"
        fi
    done
    
    green "Cleanup complete! All temporary fix scripts have been moved to $BACKUP_DIR"
    echo "You can safely delete this directory once you confirm everything is working."
else
    red "Some tests failed. Skipping cleanup to avoid removing potentially needed scripts."
    echo "Please fix the remaining issues before running cleanup."
fi

# Create a readme file for the backup directory
if [ -d "$BACKUP_DIR" ]; then
    cat > "$BACKUP_DIR/README.md" << 'EOF'
# Backup of Temporary Fix Scripts

This directory contains backup copies of the temporary fix scripts that were used to troubleshoot and fix deployment issues on the latency.space server.

These scripts are kept for reference purposes only and are no longer needed for normal operation.

You can safely delete this directory once you confirm the site is fully operational.

Scripts included:
- diagnostic.sh - Initial diagnostic script
- server-check.sh - Server status checking script
- nginx-config-fix.sh - Script to fix Nginx configuration
- filesystem-fix.sh - Script to handle read-only filesystem issues
- check-containers.sh - Script to check container status
- final-fix.sh - Comprehensive fix script
- complete-fix.sh - Final complete fix script
EOF
fi

# Print summary
echo ""
echo "===================================================="
echo "                   CLEANUP SUMMARY                   "
echo "===================================================="
echo "Main site (latency.space): $([ "$TESTS_PASSED" = true ] && echo "Working" || echo "Not working")"
echo "Mars subdomain: $([ "$TESTS_PASSED" = true ] && echo "Working" || echo "Not working")"
echo "Status subdomain: $([ "$TESTS_PASSED" = true ] && echo "Working" || echo "Not working")"
echo "Scripts cleaned up: $([ "$TESTS_PASSED" = true ] && echo "Yes" || echo "No - some tests failed")"
echo "Backup location: $BACKUP_DIR"
echo "===================================================="