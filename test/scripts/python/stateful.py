#!/usr/bin/env python3

##################
# Script imports #
##################

import datetime
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
STATE_FILE = os.environ.get("STATE_FILE", SCRIPT_NAME.replace(".py", ".state"))
COUNTER_FILE = os.environ.get("COUNTER_FILE", SCRIPT_NAME.replace(".py", ".counter"))
VAR1 = os.environ.get("VAR1", "default_value_1")
VAR2 = os.environ.get("VAR2", "default_value_2")
# Get the current time once for consistent use.
now = datetime.datetime.now()
# Default to YYYY_MM_DD_EPOCH-IN-MS
RUN_ID = os.environ.get("RUN_ID", f"{now.strftime('%Y_%m_%d')}_{int(now.timestamp() * 1000)}")
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
print(f"{LB}VAR1{NC} = {LG}{VAR1}{NC}")
print(f"{LB}VAR2{NC} = {LG}{VAR2}{NC}")

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

# 4 - Write state file ONLY on success
if exit_code == 0:
    state_path = os.path.join(METADATA_DIR, STATE_FILE)
    print(f"{LB}WRITING STATE TO '{LG}{state_path}{LB}'...{NC}")
    with open(state_path, "w") as f:
        f.write(f"VAR1={VAR1}\nVAR2={VAR2}\nrun_id={RUN_ID}\n")

# 5 - Exit after completion
print(f"{LB}### EXITING WITH EXIT CODE {LG}{exit_code}{LB} ###{NC}")
sys.exit(exit_code)
