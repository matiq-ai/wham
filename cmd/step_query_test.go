package cmd_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStepGet_AllJsonOutput verifies that `step get all -o json` produces a correct
// JSON array of all step configurations.
func TestStepGet_AllJsonOutput(t *testing.T) {
	const configPath = "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)

	outputStr, err := runWhamCommand(t, "--config", configPath, "step", "get", "all", "-o", "json")
	assert.NoError(t, err, "step get all should succeed.")

	var steps []TestStep
	err = json.Unmarshal([]byte(outputStr), &steps)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output from 'step get all'.")

	assert.Len(t, steps, 6, "There should be 6 steps in the configuration.")
	assert.Equal(t, "stateful_sh_succeed", steps[0].Name, "The first step in the config should be correctly identified.")
}

// TestStepDescribe_Single verifies that `step describe` produces a readable,
// non-empty output for a single step.
func TestStepDescribe_Single(t *testing.T) {
	const configPath = "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)

	// Run the step first to ensure it has a state to describe.
	_, err := runWhamCommand(t, "--config", configPath, "run", "stateful_sh_succeed")
	assert.NoError(t, err, "Initial run should succeed.")

	outputStr, err := runWhamCommand(t, "--config", configPath, "step", "describe", "stateful_sh_succeed")
	assert.NoError(t, err, "step describe should succeed.")

	assert.Contains(t, outputStr, "Name: stateful_sh_succeed", "Output should contain the step name.")
	assert.Contains(t, outputStr, "Configuration:", "Output should contain the Configuration section header.")
	assert.Contains(t, outputStr, "State:", "Output should contain the State section header.")
	assert.Contains(t, outputStr, "Last Action", "Output should contain state information like Last Action.")
}
