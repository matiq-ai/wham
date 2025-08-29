#!/bin/bash
# This script is used to test environment variable templating.
# It prints the values of variables that are expected to be set by WHAM.

echo "REQUIRED_VAR=${REQUIRED_VAR}"
echo "OPTIONAL_VAR_PRESENT=${OPTIONAL_VAR_PRESENT}"
echo "OPTIONAL_VAR_MISSING=${OPTIONAL_VAR_MISSING}"
echo "OPTIONAL_VAR_WITH_DEFAULT=${OPTIONAL_VAR_WITH_DEFAULT}"
