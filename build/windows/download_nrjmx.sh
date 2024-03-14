#!/usr/bin/env bash
set -e

# Gets last version of nrjmx MSI installer

echo "Downlading last version of nrjmx MSI installer"
if [[ -z $NRJMX_URL ]]; then
    NRJMX_VERSION=$(curl --silent "https://api.github.com/repos/newrelic/nrjmx/releases/latest" | grep '"tag_name":' |  grep -oE '[0-9.?]+')
    echo "Using latest nrjmx version $NRJMX_VERSION."
    NRJMX_URL=https://github.com/newrelic/nrjmx/releases/download/v$NRJMX_VERSION/nrjmx-amd64.$NRJMX_VERSION.msi
fi

curl -LSs --fail "$NRJMX_URL" -o "build/package/windows/bundle/nrjmx-amd64.msi"
