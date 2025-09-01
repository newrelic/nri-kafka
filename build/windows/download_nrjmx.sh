#!/usr/bin/env bash
set -e

# Gets last version of nrjmx MSI installer with staging support

# Set default download base URL (can be overridden with environment variable)
DOWNLOAD_BASE_URL=${DOWNLOAD_BASE_URL:-"http://nr-downloads-ohai-staging.s3-website-us-east-1.amazonaws.com"}

echo "Downloading last version of nrjmx MSI installer from $DOWNLOAD_BASE_URL"
if [[ -z $NRJMX_URL ]]; then
    # Use GitHub API to get latest version (this should work regardless of staging/production)
    NRJMX_VERSION=$(curl --silent "https://api.github.com/repos/newrelic/nrjmx/releases/latest" | grep '"tag_name":' |  grep -oE '[0-9.?]+')
    echo "Using latest nrjmx version $NRJMX_VERSION."
    
    # Construct URL using the base URL
    NRJMX_URL=$DOWNLOAD_BASE_URL/infrastructure_agent/binaries/windows/nrjmx/nrjmx-amd64.$NRJMX_VERSION.msi
fi

curl -LSs --fail "$NRJMX_URL" -o "build/package/windows/bundle/nrjmx-amd64.msi"