package cmd_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- Validation Tests with JSON Output ---

// TestValidate_Success verifies that a valid step produces a successful validation result.
func TestValidate_Success(t *testing.T) {
	const configPath = "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	outputStr, err := runWhamCommand(t, "--config", configPath, "validate", "stateful_sh_succeed", "-o", "json")

	assert.NoError(t, err, "The validate command should always exit successfully.")

	var result TestValidationResult
	err = json.Unmarshal([]byte(outputStr), &result)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output.")

	assert.True(t, result.Valid, "The 'valid' field should be true for a valid step.")
	assert.Equal(t, "all checks ok", result.Reason, "The reason should be 'all checks ok' for a valid step.")
	assert.Equal(t, "stateful_sh_succeed", result.StepName, "The step name should match the target.")
}

// TestValidate_FailNotExecutable tests that a step with a non-executable script fails validation.
func TestValidate_FailNotExecutable(t *testing.T) {
	const configPath = "../test/settings/settings_fail_not_executable.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	outputStr, err := runWhamCommand(t, "--config", configPath, "validate", "fail_script_not_executable", "-o", "json")

	assert.NoError(t, err, "The validate command should always exit successfully.")

	var result TestValidationResult
	err = json.Unmarshal([]byte(outputStr), &result)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output.")

	assert.False(t, result.Valid, "The 'valid' field should be false.")
	assert.Contains(t, result.Reason, "is not executable", "The reason should indicate the script is not executable.")
}

// TestValidate_FailScriptNotFound tests that a step with a non-existent script fails validation.
func TestValidate_FailScriptNotFound(t *testing.T) {
	const configPath = "../test/settings/settings_fail_not_found.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	outputStr, err := runWhamCommand(t, "--config", configPath, "validate", "fail_script_not_found", "-o", "json")

	assert.NoError(t, err, "The validate command should always exit successfully.")

	var result TestValidationResult
	err = json.Unmarshal([]byte(outputStr), &result)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output.")

	assert.False(t, result.Valid, "The 'valid' field should be false.")
	assert.Contains(t, result.Reason, "not found", "The reason should indicate the script was not found.")
}

// TestValidate_FailNonExistentStep tests that validating a non-existent step fails correctly.
func TestValidate_FailNonExistentStep(t *testing.T) {
	const configPath = "../test/settings/settings_ok.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	outputStr, err := runWhamCommand(t, "--config", configPath, "validate", "non_existent_step", "-o", "json")
	assert.NoError(t, err, "The validate command should always exit successfully.")

	var result TestValidationResult
	err = json.Unmarshal([]byte(outputStr), &result)
	assert.NoError(t, err, "Should be able to unmarshal the JSON output.")

	assert.False(t, result.Valid, "The 'valid' field should be false for a non-existent step.")
	assert.Equal(t, "not found in configuration", result.Reason, "The reason should indicate the step was not found.")
}
