#!/bin/bash
set -e
#
#
# Gets last version of nrjmx.jar and nrjmx.bat
#
#
echo "Downlading last version of nrjmx.jar and nrjmx.bat"
if [[ -z $NRJMX_VERSION ]]; then
    NRJMX_VERSION=$(curl --silent "https://api.github.com/repos/newrelic/nrjmx/releases/latest" | grep '"tag_name":' |  sed -E 's/.*"([^"]+)".*/\1/' | cut -d v -f2)
    echo "Using latest nrjmx version $NRJMX_VERSION."
    echo "Last version known to work is 1.6.1."
fi

curl -SL https://github.com/newrelic/nrjmx/releases/download/v${NRJMX_VERSION}/nrjmx-${NRJMX_VERSION}.tar.gz | tar xz
cp nrjmx-${NRJMX_VERSION}/lib/nrjmx-${NRJMX_VERSION}.jar nrjmx.jar
cp nrjmx-${NRJMX_VERSION}/bin/nrjmx.bat nrjmx.bat
