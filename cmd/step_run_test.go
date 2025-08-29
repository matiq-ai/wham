package cmd_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// findAndUnmarshalRunSummary finds the start of a JSON array in the `run all` command output
// and unmarshals it into the provided slice of TestStepState.
func findAndUnmarshalRunSummary(t *testing.T, outputStr string, target *[]TestStepState) {
	t.Helper()
	// The `run all` command prints a success message before the JSON,
	// so we need to find the start of the JSON array.
	jsonStartIndex := strings.Index(outputStr, "[")
	if jsonStartIndex == -1 {
		t.Fatalf("Could not find start of JSON array in output: %s", outputStr)
	}
	jsonOutput := outputStr[jsonStartIndex:]

	err := json.Unmarshal([]byte(jsonOutput), target)
	assert.NoError(t, err, "Should be able to unmarshal the JSON summary.")
}

// TestRunAll_Success tests the "happy path" using settings_ok.yaml.
// It verifies that the workflow completes successfully and the output is correct.
func TestRunAll_Success(t *testing.T) {
	configPath := "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// Run the command and request JSON output for robust testing.
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "all", "-o", "json")

	// Assertions to verify the outcome.
	assert.NoError(t, err, "The command should execute successfully without an error exit code.")

	// The output is now the JSON summary. Unmarshal it.
	var states []TestStepState
	findAndUnmarshalRunSummary(t, outputStr, &states)

	// Create a map for easy lookup
	statesMap := make(map[string]TestStepState)
	for _, s := range states {
		statesMap[s.StepName] = s
	}

	// Perform precise assertions on the data.
	assert.Len(t, states, 6, "There should be 6 steps in the summary.")
	assert.Equal(t, "run", statesMap["stateful_sh_succeed"].RunAction, "stateful_sh_succeed should have action 'run'.")
	assert.Equal(t, "run", statesMap["final_aggregator_step"].RunAction, "final_aggregator_step should have action 'run'.")
	assert.NotEmpty(t, statesMap["stateful_sh_succeed"].RunID, "A stateful step that ran should have a non-empty run_id.")
	assert.NotZero(t, statesMap["stateful_sh_succeed"].Elapsed, "A step that ran should have a non-zero elapsed time.")
}

// TestRunAll_FailRuntimeHalt tests that a workflow correctly halts when a critical
// step (can_fail: false) fails during execution.
func TestRunAll_FailRuntimeHalt(t *testing.T) {
	configPath := "../test/settings/settings_fail_runtime_halt.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "all")

	// We expect an error because a critical step fails.
	assert.Error(t, err, "The command should fail with an error exit code.")
	// Check that the resilient step was handled correctly.
	assert.Contains(t, outputStr, "⚠️ Step 'resilient_step_fails' failed but continuing", "The resilient step should fail but not halt the workflow.")
	// Check that the critical step's failure is reported.
	assert.Contains(t, outputStr, "step 'critical_step_fails' failed", "The error message for the critical failing step should be present.")
	// Check that the workflow did not complete.
	assert.NotContains(t, outputStr, "All steps completed successfully.", "The final success message should not be present.")
}

// TestForceSingle_InjectsParam tests that forcing a step correctly injects the 'force'
// parameter via runtime templating.
func TestForceSingle_InjectsParam(t *testing.T) {
	configPath := "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// We force a step that has both shared and local args defined in settings_ok.yaml
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "stateless_with_params", "--force")

	// We expect the command to succeed.
	assert.NoError(t, err, "The command should execute successfully.")
	assert.Contains(t, outputStr, "Step 'stateless_with_params' completed successfully.", "The success message for the specific step should be present.")
	// This is the key assertion: check that the script's output shows the injected 'force' parameter.
	assert.Contains(t, outputStr, "CLI PARAMETERS = force --local-param value", "The output should show the 'force' parameter injected alongside local params, now defined as a slice.")
}

// TestRunSingle_InjectsEnvVars verifies that environment variables defined
// in the config are correctly injected into the script's environment.
func TestRunSingle_InjectsEnvVars(t *testing.T) {
	configPath := "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// We run a step that has specific env_vars defined.
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "stateful_sh_succeed")

	// We expect the command to succeed.
	assert.NoError(t, err, "The command should execute successfully.")
	assert.Contains(t, outputStr, "Step 'stateful_sh_succeed' completed successfully.", "The success message for the specific step should be present.")

	// Key assertions: check that the script's output shows the injected environment variables.
	assert.Contains(t, outputStr, "VAR1 = injected_value_1", "The output should show the injected value for VAR1.")
	assert.Contains(t, outputStr, "VAR2 = 22", "The output should show the injected value for VAR2.")
}

// TestRun_EnvTemplating_Success verifies that `getenv` and `require_env` correctly
// process environment variables. It checks for successful retrieval, handling of
// missing optional variables, and correct application of default values.
func TestRun_EnvTemplating_Success(t *testing.T) {
	configPath := "../test/settings/settings_env_templating_ok.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// Set an environment variable for the test to find.
	t.Setenv("TEST_VAR_PRESENT", "value_is_here")

	// Run the step that uses the template functions.
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "test_env_vars_ok")

	// Assertions
	assert.NoError(t, err, "The command should execute successfully.")
	assert.Contains(t, outputStr, "Step 'test_env_vars_ok' completed successfully.", "The success message for the specific step should be present.")

	// Check that the script's output shows the correctly templated environment variables.
	assert.Contains(t, outputStr, "REQUIRED_VAR=value_is_here", "require_env should have found the variable.")
	assert.Contains(t, outputStr, "OPTIONAL_VAR_PRESENT=value_is_here", "getenv should have found the variable.")
	assert.Contains(t, outputStr, "OPTIONAL_VAR_MISSING=", "getenv should return an empty string for a missing variable without a default.")
	assert.Contains(t, outputStr, "OPTIONAL_VAR_WITH_DEFAULT=this_is_the_default", "getenv should use the default value for a missing variable.")
}

// TestRun_EnvTemplating_Failure verifies that `require_env` correctly
// fails the step if a mandatory environment variable is missing.
func TestRun_EnvTemplating_Failure(t *testing.T) {
	configPath := "../test/settings/settings_env_templating_fail.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// Run the step that uses require_env on a missing variable.
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "test_require_env_fails")

	// Assertions
	assert.Error(t, err, "The command should fail with an error exit code.")
	assert.Contains(t, outputStr, "failed to process template for env_var 'REQUIRED_VAR_MISSING'", "Error message should specify the failing env_var.")
	assert.Contains(t, outputStr, "required environment variable 'TEST_VAR_THAT_DOES_NOT_EXIST' is not set or is empty", "Error message should specify the missing environment variable.")
}

// TestRunAll_Force verifies that `run all --force` correctly re-executes all steps,
// including those that would normally be skipped.
func TestRunAll_Force(t *testing.T) {
	const configPath = "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// First, run the workflow normally to establish a baseline state.
	_, err := runWhamCommand(t, "--config", configPath, "run", "all")
	assert.NoError(t, err, "The initial run should succeed.")

	// Now, run the workflow again with --force and JSON output.
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "all", "--force", "-o", "json")
	assert.NoError(t, err, "The forced run should succeed.")

	var states []TestStepState
	findAndUnmarshalRunSummary(t, outputStr, &states)

	// The key assertion: check the action for each step.
	for _, s := range states {
		if s.StepName == "stateless_sh_maybe_fail" {
			// This step can either run successfully or fail, both are acceptable outcomes on a forced run.
			assert.Contains(t, []string{"run", "failed"}, s.RunAction,
				"Step '%s' should have action 'run' or 'failed' on a forced run, but got '%s'", s.StepName, s.RunAction)
		} else {
			// All other steps must have the action "run" because it's a forced run.
			assert.Equal(t, "run", s.RunAction,
				"Step '%s' should have action 'run' on a forced run, not '%s'", s.StepName, s.RunAction)
		}
	}
}

// TestRunAll_FromToFlags verifies the correct behavior of the --from and --to flags.
func TestRunAll_FromToFlags(t *testing.T) {
	const configPath = "../test/settings/settings_from_to_flags.yaml"

	testCases := []struct {
		name             string
		args             []string
		preRunStep       string // A step to run beforehand to satisfy preconditions
		expectError      bool
		errorContains    string
		expectedToRun    []string
		expectedToNotRun []string
	}{
		{
			name:             "run from step",
			args:             []string{"run", "all", "--from", "step-b"},
			preRunStep:       "step-a", // Pre-run to satisfy dependency
			expectError:      false,
			expectedToRun:    []string{"step-b", "step-c", "step-d"},
			expectedToNotRun: []string{"step-a"},
		},
		{
			name:             "run to step",
			args:             []string{"run", "all", "--to", "step-c"},
			expectError:      false,
			expectedToRun:    []string{"step-a", "step-b", "step-c"},
			expectedToNotRun: []string{"step-d"},
		},
		{
			name:             "run between from and to",
			args:             []string{"run", "all", "--from", "step-b", "--to", "step-c"},
			preRunStep:       "step-a",
			expectError:      false,
			expectedToRun:    []string{"step-b", "step-c"},
			expectedToNotRun: []string{"step-a", "step-d"},
		},
		{
			name:          "fail with from on single step",
			args:          []string{"run", "step-a", "--from", "step-a"},
			expectError:   true,
			errorContains: "--from and --to flags can only be used with the 'all' target",
		},
		{
			name:          "fail with non-existent from step",
			args:          []string{"run", "all", "--from", "non_existent_step"},
			expectError:   true,
			errorContains: "step specified in --from not found: 'non_existent_step'",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanTestStates(t, configPath)
			t.Cleanup(func() { cleanTestStates(t, configPath) })

			if tc.preRunStep != "" {
				_, err := runWhamCommand(t, "--config", configPath, "run", tc.preRunStep)
				assert.NoError(t, err, "Pre-run step should succeed.")
			}

			argsWithConfig := append([]string{"--config", configPath}, tc.args...)
			outputStr, err := runWhamCommand(t, argsWithConfig...)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, outputStr, tc.errorContains)
				return
			}

			assert.NoError(t, err)

			for _, stepName := range tc.expectedToRun {
				assert.Contains(t, outputStr, "Running step '"+stepName+"'", "Expected step '%s' to run, but it didn't.", stepName)
			}
			for _, stepName := range tc.expectedToNotRun {
				assert.NotContains(t, outputStr, "Running step '"+stepName+"'", "Expected step '%s' not to run, but it did.", stepName)
			}
		})
	}
}

// TestRunSingle_Success tests running a single, valid source node step.
func TestRunSingle_Success(t *testing.T) {
	configPath := "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// We run a source node, which should always be runnable on a clean slate.
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "stateful_sh_succeed")

	// We expect the command to succeed.
	assert.NoError(t, err, "The command should execute successfully.")
	assert.Contains(t, outputStr, "Step 'stateful_sh_succeed' completed successfully.", "The success message for the specific step should be present.")
	assert.NotContains(t, outputStr, "final_aggregator_step", "The output should not contain logs from other steps.")
}

// TestRunSingle_FailPrecondition tests that running a single step fails if its
// critical predecessors have not been run yet.
func TestRunSingle_FailPrecondition(t *testing.T) {
	configPath := "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// We run a step whose predecessor has not run. This should fail the precondition check.
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "stateless_sh_succeed")

	// We expect an error because the precondition (predecessor state) is not met.
	assert.Error(t, err, "The command should fail with an error exit code.")
	assert.Contains(t, outputStr, "precondition check failed", "The error message should indicate a precondition failure.")
	assert.Contains(t, outputStr, "has no valid WHAM state (empty run_id)", "The error message should specify the missing state from the predecessor.")
	assert.NotContains(t, outputStr, "Execution Summary", "The execution summary should not be printed on a failure.")
}

// TestRunAll_RetrySuccess verifies that a step correctly retries and eventually succeeds.
func TestRunAll_RetrySuccess(t *testing.T) {
	configPath := "../test/settings/settings_retry_success.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// Run the workflow. We don't need to check its output, just that it succeeds.
	_, err := runWhamCommand(t, "--config", configPath, "run", "all")
	assert.NoError(t, err, "The command should succeed after retries.")

	// Now, check the final state to confirm the outcome. This is more robust
	// than parsing log messages.
	outputStr, err := runWhamCommand(t, "--config", configPath, "state", "get", "all", "-o", "json")
	assert.NoError(t, err, "state get all should succeed.")

	var states []TestStepState
	err = json.Unmarshal([]byte(outputStr), &states)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output from 'state get all'.")

	// Find the specific step and check its state.
	var retryStepState TestStepState
	for _, s := range states {
		if s.StepName == "retry_step_succeeds" {
			retryStepState = s
			break
		}
	}

	assert.Equal(t, "retry_step_succeeds", retryStepState.StepName, "The retry step should be found in the state summary.")
	assert.Equal(t, "run", retryStepState.RunAction, "The final action for the retried step should be 'run'.")
}

// TestRunAll_RetryExhaustedFail verifies that a workflow halts when a step fails after exhausting all retries.
func TestRunAll_RetryExhaustedFail(t *testing.T) {
	configPath := "../test/settings/settings_retry_fail.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "all")

	assert.Error(t, err, "The command should fail as retries are exhausted.")
	// Check for the two attempts that were made
	assert.Contains(t, outputStr, "Simulating failure attempt #1", "Should show the first failure message.")
	assert.Contains(t, outputStr, "Simulating failure attempt #2", "Should show the second failure message.")
	// Check that it did NOT succeed
	assert.NotContains(t, outputStr, "Simulating success", "The script should not succeed.")
	// Check for the final error message from WHAM
	assert.Contains(t, outputStr, "step 'retry_step_failure' failed", "WHAM should report the final failure.")
	assert.NotContains(t, outputStr, "All steps completed successfully.", "The workflow should not complete successfully.")
}

// TestWorkDir_ChangesDirectory verifies that the `work_dir` flag
// correctly changes the script's current working directory.
func TestWorkDir_ChangesDirectory(t *testing.T) {
	configPath := "../test/settings/settings_workdir.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// Test with work_dir set to the script's directory.
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "workdir_set_to_script_dir")

	assert.NoError(t, err)
	// The script should run inside its own directory.
	assert.Contains(t, outputStr, "My current working directory is: ", "The script should print its CWD.")
	assert.Contains(t, outputStr, "test/scripts/bash", "The CWD should be the script's directory.") // Note: this is a relative path check

	// Test with work_dir not set (should run in project root).
	outputStr, err = runWhamCommand(t, "--config", configPath, "run", "workdir_not_set")

	assert.NoError(t, err)
	// The script should run from the WHAM process's CWD (the project root).
	assert.Contains(t, outputStr, "My current working directory is: ", "The script should print its CWD.")
	assert.NotContains(t, outputStr, "test/scripts/bash", "The CWD should NOT be the script's directory.")
	// Get the project root to check against.
	projectRoot, _ := filepath.Abs("..") // Get the absolute path of the parent directory (project root)
	assert.Contains(t, outputStr, projectRoot, "The CWD should be the project root.")
}

// TestWorkDir_FailsOnInvalidPath verifies that the step fails if `work_dir` is not a valid directory.
func TestWorkDir_FailsOnInvalidPath(t *testing.T) {
	configPath := "../test/settings/settings_fail_workdir.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// Run the step with the invalid work_dir from settings_fail_workdir.yaml
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "fail_workdir_not_found")

	// We expect an error because the directory does not exist.
	assert.Error(t, err, "The command should fail with an error exit code.")
	assert.Contains(t, outputStr, "invalid work_dir './non_existent_dir' for step 'fail_workdir_not_found'", "The error message should indicate an invalid work_dir.")
	assert.Contains(t, outputStr, "path does not exist or is not a directory", "The error message should be specific about the cause.")
}
