#!/usr/bin/env python3

##################
# Script imports #
##################

import os
import random
import sys

###########################
# Script global variables #
###########################

# Output colors
if os.environ.get("NO_COLOR"):
    LR = ''
    LG = ''
    LB = ''
    NC = ''
else:
    LR = '\033[1;31m'  # light red
    LG = '\033[1;32m'  # light green
    LB = '\033[1;34m'  # light blue
    NC = '\033[0m'     # no color

# Initialize global variables
WORK_DIR = os.path.dirname(os.path.abspath(__file__))
SCRIPT_NAME = os.path.basename(__file__)

# Use WHAM-injected directories if available, otherwise fall back to defaults for standalone testing
DATA_DIR = os.environ.get("VAR_DATA_DIR", os.path.abspath(os.path.join(WORK_DIR, "../../states/data")))
METADATA_DIR = os.environ.get("VAR_METADATA_DIR", os.path.abspath(os.path.join(WORK_DIR, "../../states/metadata")))

# Allow injecting variable values, with sensible defaults
COUNTER_FILE = os.environ.get("COUNTER_FILE", SCRIPT_NAME.replace(".py", ".counter"))
SIMULATE_FAIL_COUNT = int(os.environ.get("SIMULATE_FAIL_COUNT", "0")) # variable for retry simulation
EXIT_STATUS = os.environ.get("EXIT_STATUS", "success")  # default to success if not set

#####################
# Script operations #
#####################

# 1 - Ensure metadata directory exists
if not os.path.isdir(METADATA_DIR):
    print(f"{LR}### ERROR: Metadata directory {METADATA_DIR} does not exist!{NC}")
    sys.exit(1)

# 2 - Print step info
print(f"{LB}### STARTING '{LG}{SCRIPT_NAME}{LB}' ###{NC}")
print(f"{LB}DATA_DIR{NC} = {LG}{DATA_DIR}{NC}")
print(f"{LB}METADATA_DIR{NC} = {LG}{METADATA_DIR}{NC}")
print(f"{LB}CLI PARAMETERS{NC} = {LG}{' '.join(sys.argv[1:])}{NC}")

# 3 - Determine exit code
exit_code = 0
if SIMULATE_FAIL_COUNT > 0:
    # This logic simulates a script that fails N times before succeeding
    counter_path = os.path.join(METADATA_DIR, COUNTER_FILE)
    current_attempt = 0
    try:
        with open(counter_path, "r") as f:
            current_attempt = int(f.read().strip())
    except (FileNotFoundError, ValueError):
        current_attempt = 0

    with open(counter_path, "w") as f:
        f.write(str(current_attempt + 1))

    if current_attempt < SIMULATE_FAIL_COUNT:
        exit_code = 1
        print(f"{LR}### Simulating failure attempt #{current_attempt + 1} (will succeed after {SIMULATE_FAIL_COUNT} failures) ###{NC}")
    else:
        print(f"{LG}### Simulating success after {SIMULATE_FAIL_COUNT} failures ###{NC}")
        exit_code = 0
elif EXIT_STATUS == "random":
    exit_code = random.randint(0, 1)
elif EXIT_STATUS == "fail":
    exit_code = 1

# 4 - Stateless: do not write state file
# EMPTY

# 5 - Exit after completion
print(f"{LB}### EXITING WITH EXIT CODE {LG}{exit_code}{LB} ###{NC}")
sys.exit(exit_code)
