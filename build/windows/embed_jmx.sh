#!/bin/bash
set -e
#
#
# Gets last version of nrjmx.jar and nrjmx.bat
#
#
curl https://raw.githubusercontent.com/newrelic/nrjmx/master/bin/nrjmx.bat --output nrjmx.bat
latest_tag=$(curl --silent "https://api.github.com/repos/${JMX_REPO}/releases/latest" | grep '"tag_name":' |  sed -E 's/.*"([^"]+)".*/\1/' | cut -d v -f2)
curl -SL http://download.newrelic.com/infrastructure_agent/binaries/linux/noarch/nrjmx_linux_${latest_tag}_noarch.tar.gz | tar xz; cp usr/bin/nrjmx.jar nrjmx.jar