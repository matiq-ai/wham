package cmd_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStateDelete_Single verifies that deleting a single step's state works correctly
// and produces the correct structured output.
func TestStateDelete_Single(t *testing.T) {
	const configPath = "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)

	// First, run a step to create a state file.
	_, err := runWhamCommand(t, "--config", configPath, "run", "stateful_sh_succeed")
	assert.NoError(t, err, "Initial run should succeed.")

	// Now, delete the state using the --yes flag to bypass the interactive prompt,
	// and get structured output.
	outputStr, err := runWhamCommand(t, "--config", configPath, "state", "delete", "stateful_sh_succeed", "--yes", "-o", "json")
	assert.NoError(t, err, "State deletion should succeed.")

	var result TestDeletionResult
	err = json.Unmarshal([]byte(outputStr), &result)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output.")

	assert.Equal(t, "deleted", result.Status, "The status should be 'deleted'.")
	assert.Equal(t, "stateful_sh_succeed", result.StepName, "The step name should match.")
}

// TestStateGet_AllJsonOutput verifies that `state get all -o json` produces a correct
// JSON array of all step states after a full run.
func TestStateGet_AllJsonOutput(t *testing.T) {
	const configPath = "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)

	// First, run the entire workflow to generate states for all steps.
	_, err := runWhamCommand(t, "--config", configPath, "run", "all")
	assert.NoError(t, err, "The initial 'run all' should succeed.")

	// Now, get the state of all steps in JSON format.
	outputStr, err := runWhamCommand(t, "--config", configPath, "state", "get", "all", "-o", "json")
	assert.NoError(t, err, "state get all should succeed.")

	var states []TestStepState
	err = json.Unmarshal([]byte(outputStr), &states)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output from 'state get all'.")

	assert.Len(t, states, 6, "There should be 6 steps in the summary.")
	assert.Equal(t, "run", states[0].RunAction, "The first step in the summary should have action 'run'.")
}

// TestStateDelete_AllWithYesFlag verifies that `state delete all --yes` works
// non-interactively and produces the correct structured output.
func TestStateDelete_AllWithYesFlag(t *testing.T) {
	const configPath = "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)

	// First, run the workflow to create state files.
	_, err := runWhamCommand(t, "--config", configPath, "run", "all")
	assert.NoError(t, err, "Initial 'run all' should succeed.")

	// Now, delete all states using the --yes flag and request JSON output.
	outputStr, err := runWhamCommand(t, "--config", configPath, "state", "delete", "all", "--yes", "-o", "json")
	assert.NoError(t, err, "state delete all --yes should succeed.")

	var results []TestDeletionResult
	err = json.Unmarshal([]byte(outputStr), &results)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output from 'state delete all'.")

	assert.Len(t, results, 6, "Should receive deletion results for all 6 steps.")
	assert.Equal(t, "deleted", results[0].Status, "The status for the first step should be 'deleted'.")
}
