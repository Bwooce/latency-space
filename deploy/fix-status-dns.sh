#!/bin/bash
# This script previously fixed the status.latency.space DNS record
# The status subdomain has been removed, as status information is now integrated with the main site

# Colors for better output
red() { echo -e "\033[0;31m$1\033[0m"; }
green() { echo -e "\033[0;32m$1\033[0m"; }
blue() { echo -e "\033[0;34m$1\033[0m"; }
yellow() { echo -e "\033[0;33m$1\033[0m"; }

blue "🌐 Status Subdomain DNS Configuration"
yellow "⚠️  Notice: The status.latency.space subdomain has been removed"
green "✅ Status information is now integrated with the main latency.space website"
echo ""
echo "No DNS changes are needed."

exit 0