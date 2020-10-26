#!/bin/bash
set -e
#
#
# Gets last version of nrjmx.jar and nrjmx.bat
#
#
echo "Downlading last version of nrjmx.jar and nrjmx.bat"
latest_tag=$(curl --silent "https://api.github.com/repos/newrelic/nrjmx/releases/latest" | grep '"tag_name":' |  sed -E 's/.*"([^"]+)".*/\1/' | cut -d v -f2)
curl -SL https://github.com/newrelic/nrjmx/releases/download/v${latest_tag}/nrjmx-${latest_tag}.tar.gz | tar xz
cp nrjmx-${latest_tag}/lib/nrjmx-${latest_tag}.jar nrjmx.jar
cp nrjmx-${latest_tag}/bin/nrjmx.bat nrjmx.bat