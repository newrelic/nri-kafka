#!/bin/bash
set -e
#
#
# Gets dist/zip_dirty created by Goreleaser and reorganize inside files
#
#
PROJECT_PATH=$1

for zip_dirty in $(find dist -regex ".*_dirty\.\(zip\)");do
  zip_file_name=${zip_dirty:5:${#zip_dirty}-(5+10)} # Strips begining and end chars
  ZIP_CLEAN="${zip_file_name}.zip"
  ZIP_TMP="dist/zip_temp"
  ZIP_CONTENT_PATH="${ZIP_TMP}/${zip_file_name}_content"

  mkdir -p "${ZIP_CONTENT_PATH}"

  ls -la "${zip_dirty}"

  AGENT_DIR_IN_ZIP_PATH="${ZIP_CONTENT_PATH}/New Relic/newrelic-infra/newrelic-integrations/"
  CONF_IN_ZIP_PATH="${ZIP_CONTENT_PATH}/New Relic/newrelic-infra/integrations.d/"

  mkdir -p "${AGENT_DIR_IN_ZIP_PATH}/bin"
  mkdir -p "${CONF_IN_ZIP_PATH}"

  echo "===> Decompress ${zip_file_name} in ${ZIP_CONTENT_PATH}"
  unzip ${zip_dirty} -d ${ZIP_CONTENT_PATH}

  echo "===> Move files inside ${zip_file_name}"
  mv ${ZIP_CONTENT_PATH}/nri-${INTEGRATION}.exe "${AGENT_DIR_IN_ZIP_PATH}/bin"
  mv ${ZIP_CONTENT_PATH}/${INTEGRATION}-win-definition.yml "${AGENT_DIR_IN_ZIP_PATH}"
  mv ${ZIP_CONTENT_PATH}/${INTEGRATION}-win-config.yml.sample "${CONF_IN_ZIP_PATH}"

  echo "===> Embeding nrjmx"
  JMX_IN_ZIP_PATH="${ZIP_CONTENT_PATH}/New Relic/nrjmx/"
  mkdir -p "${JMX_IN_ZIP_PATH}"
  JMX_REPO="newrelic/nrjmx"
  curl https://raw.githubusercontent.com/newrelic/nrjmx/master/bin/nrjmx.bat --output "${JMX_IN_ZIP_PATH}/nrjmx.bat"
  #latest_jmx_version=$(curl --silent "https://api.github.com/repos/${JMX_REPO}/releases/latest" | grep '"tag_name":' |  sed -E 's/.*"([^"]+)".*/\1/' | cut -d v -f2)
  curl -SL "http://download.newrelic.com/infrastructure_agent/binaries/linux/noarch/nrjmx_linux_${NRJMX_VERSION}_noarch.tar.gz" | tar xz; cp usr/bin/nrjmx.jar "${JMX_IN_ZIP_PATH}/nrjmx-${NRJMX_VERSION}.jar"

  echo "===> Show jmx files"
  echo "JMX_IN_ZIP_PATH = $JMX_IN_ZIP_PATH"
  ls -la "${JMX_IN_ZIP_PATH}"

  echo "===> Creating zip ${ZIP_CLEAN}"
  cd "${ZIP_CONTENT_PATH}"
  zip -r ../${ZIP_CLEAN} .
  cd $PROJECT_PATH
  echo "===> Moving zip ${ZIP_CLEAN}"
  mv "${ZIP_TMP}/${ZIP_CLEAN}" dist/
  echo "===> Cleaning dirty zip ${zip_dirty}"
  rm "${zip_dirty}"
done