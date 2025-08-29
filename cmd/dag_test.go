package cmd_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDAGGet_TableOutput verifies that `dag get` produces a readable,
// correctly sorted table output.
func TestDAGGet_TableOutput(t *testing.T) {
	configPath := "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)

	outputStr, err := runWhamCommand(t, "--config", configPath, "dag", "get")

	assert.NoError(t, err, "The command should execute successfully.")
	// Check for headers
	assert.Contains(t, outputStr, "DEPTH", "Output should contain DEPTH header.")
	assert.Contains(t, outputStr, "NAME", "Output should contain NAME header.")
	assert.Contains(t, outputStr, "PREDECESSORS", "Output should contain PREDECESSORS header.")
	// Check for a specific step to ensure content is rendered
	assert.Contains(t, outputStr, "final_aggregator_step", "Output should contain a known step name.")
}

// TestDAGGet_JsonOutput verifies that `dag get -o json` produces a valid JSON
// array with the correct structure and data.
func TestDAGGet_JsonOutput(t *testing.T) {
	configPath := "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)

	outputStr, err := runWhamCommand(t, "--config", configPath, "dag", "get", "-o", "json")

	assert.NoError(t, err, "The command should execute successfully.")

	var dagInfo []TestDAGStepInfo
	err = json.Unmarshal([]byte(outputStr), &dagInfo)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output into the DAGStepInfo struct.")

	// Verify the content of the unmarshaled data.
	assert.Len(t, dagInfo, 6, "The DAG should contain 6 steps.")

	// Find a specific step for a more detailed assertion.
	var finalStep TestDAGStepInfo
	for _, step := range dagInfo {
		if step.Name == "final_aggregator_step" {
			finalStep = step
			break
		}
	}

	assert.Equal(t, "final_aggregator_step", finalStep.Name, "The final step should be found.")
	assert.Equal(t, 3, finalStep.Depth, "The depth of the final step should be 3.")
	assert.Contains(t, finalStep.PreviousSteps, "stateless_sh_maybe_fail", "The final step should depend on 'stateless_sh_maybe_fail'.")
}
