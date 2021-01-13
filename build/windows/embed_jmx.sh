#!/bin/bash
set -e
#
#
# Gets last version of nrjmx.jar and nrjmx.bat
#
#
echo "Fetching latest nrjmx version..."
latest_tag=$(curl --silent "https://api.github.com/repos/newrelic/nrjmx/releases/latest" | grep '"tag_name":' | grep -oE 'v([0-9]\.?)+' | sed 's/v//')

echo "Downloading nrjmx_windows_${latest_tag}_amd64.zip..."
wget --no-verbose "https://github.com/newrelic/nrjmx/releases/download/v${latest_tag}/nrjmx_windows_${latest_tag}_amd64.zip" -O nrjmx_windows.zip
# TODO(roobre): This is broken an needs fixing
unzip -p "nrjmx_windows.zip" nrjmx-${latest_tag}/bin/nrjmx.bat > nrjmx.bat

echo "Downloading nrjmx-${latest_tag}-noarch.jar..."
wget --no-verbose "https://github.com/newrelic/nrjmx/releases/download/v${latest_tag}/nrjmx-${latest_tag}-noarch.jar" -O nrjmx.jar
