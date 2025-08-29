#!/usr/bin/env bash

###########################
# Script global variables #
###########################

# Output colors
if [ -z "${NO_COLOR-}" ]; then
    LR='\033[1;31m' # light red
    LG='\033[1;32m' # light green
    LB='\033[1;34m' # light blue
    NC='\033[0m'    # no color
else
    LR=""
    LG=""
    LB=""
    NC=""
fi

# Initialize global variables
WORK_DIR="$( cd "$( dirname "$0" )" && pwd )"
SCRIPT_NAME="$(basename "$0")"

# Use WHAM-injected directories if available, otherwise fall back to defaults for standalone testing
DATA_DIR="${VAR_DATA_DIR:-${WORK_DIR}/../../states/data}"
METADATA_DIR="${VAR_METADATA_DIR:-${WORK_DIR}/../../states/metadata}"

# Allow injecting variable values, with sensible defaults
STATE_FILE="${STATE_FILE:-${SCRIPT_NAME%%.sh}.state}"
COUNTER_FILE="${COUNTER_FILE:-${SCRIPT_NAME%%.sh}.counter}"
VAR1="${VAR1:-default_value_1}"
VAR2="${VAR2:-default_value_2}"
RUN_ID="${RUN_ID:-$(date +%Y_%m_%d_%s%3N)}" # default to YYYY_MM_DD_EPOCH-IN-MS
SIMULATE_FAIL_COUNT="${SIMULATE_FAIL_COUNT:-0}" # variable for retry simulation
EXIT_STATUS="${EXIT_STATUS:-success}" # default to success if not set

#####################
# Script operations #
#####################

# 0 - Ensure that the script fails if any command fails
set -euo pipefail

# 1 - Ensure metadata directory exists
if [[ ! -d "${METADATA_DIR}" ]]; then
    printf "${LR}### ERROR: Metadata directory ${METADATA_DIR} does not exist!${NC}\n"
    exit 1
fi

# 2 - Print step info
printf "${LB}### STARTING '${LG}$0${LB}' ###${NC}\n"
printf "${LB}DATA_DIR${NC} = ${LG}${DATA_DIR}${NC}\n"
printf "${LB}METADATA_DIR${NC} = ${LG}${METADATA_DIR}${NC}\n"
printf "${LB}CLI PARAMETERS${NC} = ${LG}%s${NC}\n" "$*"
printf "${LB}VAR1${NC} = ${LG}${VAR1}${NC}\n"
printf "${LB}VAR2${NC} = ${LG}${VAR2}${NC}\n"

# 3 - Determine exit code
exit_code=0  # <- default to success
if [[ "$SIMULATE_FAIL_COUNT" -gt 0 ]]; then
    # This logic simulates a script that fails N times before succeeding
    counter_file="${METADATA_DIR}/${COUNTER_FILE}" 
    # - read the current attempt number, default to 0 if file doesn't exist or is not a number
    current_attempt=0
    if [[ -f "$counter_file" ]] && [[ "$(cat "$counter_file")" =~ ^[0-9]+$ ]]; then
        current_attempt=$(cat "$counter_file")
    fi
    # - increment and save the attempt number for the next run
    echo "$((current_attempt + 1))" > "$counter_file"
    # - fail if the current attempt number is less than the desired number of failures
    if (( current_attempt < SIMULATE_FAIL_COUNT )); then
        printf "${LR}### Simulating failure attempt #%s (will succeed after %s failures) ###${NC}\n" "$((current_attempt + 1))" "$SIMULATE_FAIL_COUNT"
        exit_code=1
    else
        printf "${LG}### Simulating success after %s failures ###${NC}\n" "$SIMULATE_FAIL_COUNT"
        exit_code=0
    fi
elif [[ "$EXIT_STATUS" == "random" ]]; then
    exit_code="$((0 + RANDOM % 2))" # <- randomly succeed or fail
elif [[ "$EXIT_STATUS" == "fail" ]]; then
    exit_code=1 # <- failure completion
fi

# 4 - Write state file ONLY on success
if [[ "$exit_code" -eq 0 ]]; then
    printf "${LB}WRITING STATE TO '${LG}${METADATA_DIR}/${STATE_FILE}${LB}'...${NC}\n"
    cat << EOF > "${METADATA_DIR}/${STATE_FILE}"
VAR1=$VAR1
VAR2=$VAR2
run_id=$RUN_ID
EOF
fi

# 5 - Exit after completion
printf "${LB}### EXITING WITH EXIT CODE ${LG}${exit_code}${LB} ###${NC}\n"
exit $exit_code
