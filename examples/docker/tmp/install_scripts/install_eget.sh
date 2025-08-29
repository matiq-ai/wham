#!/bin/sh

###########################
# Script global variables #
###########################

PACKAGE_NAME="eget"
WORK_DIR="/tmp"
DEST_DIR="/usr/local/bin"
EGET_DL_SHASUM="${EGET_DL_SHASUM:-"0e64b8a3c13f531da005096cc364ac77835bda54276fedef6c62f3dbdc1ee919"}"

####################
# Script functions #
####################

cleanup() {
  rm -f ${WORK_DIR}/${PACKAGE_NAME}*
}

#####################
# Script operations #
#####################

# Fail on any error
set -euo pipefail

# Run cleanup on exit, even on error
trap cleanup EXIT

# cd to work dir
cd "${WORK_DIR}"

# Download the latest package release
curl -fLo "${PACKAGE_NAME}.sh" "https://zyedidia.github.io/${PACKAGE_NAME}.sh"

# Check the binary against its checksum file
echo "${EGET_DL_SHASUM}  ${PACKAGE_NAME}.sh" | sha256sum -c -

# Install the package
. ./${PACKAGE_NAME}.sh
install -o root -g root -m 0755 "${PACKAGE_NAME}" "${DEST_DIR}/${PACKAGE_NAME}"

# Exit after successful completion
echo "${PACKAGE_NAME} installed successfully!"
exit 0
