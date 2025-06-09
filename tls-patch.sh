#!/bin/bash
set -e

# Script to fix TLS certificate validation issues for Seanime

echo "Starting TLS certificate patch..."

# List of problematic domains
DOMAINS=(
  "anilist.co"
  "graphql.anilist.co"
  "api.anilist.co"
  "s4.anilist.co"
  "img.anili.st"
  "media.kitsu.io"
  "nyaa.si"
  "animethemes.moe"
)

# Create a directory for certificates
mkdir -p /tmp/certs

# Download certificates from each domain
for domain in "${DOMAINS[@]}"; do
  echo "Downloading certificate for $domain..."
  echo | openssl s_client -showcerts -servername $domain -connect $domain:443 2>/dev/null | \
    sed -ne '/-BEGIN CERTIFICATE-/,/-END CERTIFICATE-/p' > "/tmp/certs/$domain.crt"
  
  # Add to CA bundle
  cat "/tmp/certs/$domain.crt" >> /etc/ssl/certs/ca-certificates.crt
  echo "Added certificate for $domain to CA bundle"
done

# Update CA certificates
update-ca-certificates

echo "TLS certificate patch completed successfully"
