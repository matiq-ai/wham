package cmd_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"text/template"

	"matiq.ai/wham/cmd"

	"github.com/stretchr/testify/assert"
)

// TestInit_FailCycle tests the DAG validation for circular dependencies.
// It verifies that the program exits with an error and prints the correct message.
func TestInit_FailCycle(t *testing.T) {
	configPath := "../test/settings/settings_fail_dag_cycle.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "all")

	// We expect an error in this case.
	assert.Error(t, err, "The command should fail with an error exit code.")
	assert.Contains(t, outputStr, "circular dependency detected", "The output should contain the specific circular dependency error message.")
	assert.NotContains(t, outputStr, "Execution Summary", "The execution summary should not be printed on a validation failure.")
}

// TestInit_FailDuplicateStepName verifies that WHAM fails to initialize if the
// configuration contains two steps with the same name.
func TestInit_FailDuplicateStepName(t *testing.T) {
	configPath := "../test/settings/settings_fail_duplicate_step.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "all")

	// We expect an error during initialization.
	assert.Error(t, err, "The command should fail with an error exit code.")
	assert.Contains(t, outputStr, "Failed to initialize WHAM engine", "The error should be from the engine initialization.")
	assert.Contains(t, outputStr, "duplicate step name found", "The error message should specify a duplicate step name.")
	assert.Contains(t, outputStr, "'duplicate_step'", "The error message should mention the duplicated step name.")
	assert.NotContains(t, outputStr, "Execution Summary", "The execution summary should not be printed on an initialization failure.")
}

// TestInit_FailNonExistentPredecessor tests that the workflow fails validation if a step
// depends on a predecessor that is not defined in the configuration.
func TestInit_FailNonExistentPredecessor(t *testing.T) {
	const configPath = "../test/settings/settings_fail_non_existent_predecessor.yaml"
	cleanTestStates(t, configPath)                       // Clean before
	t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

	// This error is caught during the topological sort, which happens for `run all`.
	outputStr, err := runWhamCommand(t, "--config", configPath, "run", "all")

	// We expect a DAG validation error.
	assert.Error(t, err, "The command should fail with an error exit code.")
	assert.Contains(t, outputStr, "declares non-existent previous step", "The error message should indicate a missing predecessor.")
	assert.NotContains(t, outputStr, "Execution Summary", "The execution summary should not be printed on a validation failure.")
}

// TestInit_FailInvalidStepDefinition verifies that WHAM fails to initialize if the
// configuration contains a semantically invalid step definition.
func TestInit_FailInvalidStepDefinition(t *testing.T) {
	testCases := []struct {
		name           string
		configFileName string
		errContains    string
	}{
		{"missing step name", "settings_fail_step_no_name.yaml", "step name cannot be empty"},
		{"missing command", "settings_fail_step_no_command.yaml", "command cannot be empty"},
		{"stateful missing state_file", "settings_fail_step_no_statefile.yaml", "must have a 'state_file' defined"},
		{"stateful missing run_id_var", "settings_fail_step_no_runidvar.yaml", "must have a 'run_id_var' defined"},
		{"negative retries", "settings_fail_step_negative_retries.yaml", "retries cannot be negative"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configPath := "../test/settings/" + tc.configFileName
			cleanTestStates(t, configPath)                       // Clean before
			t.Cleanup(func() { cleanTestStates(t, configPath) }) // Clean after

			outputStr, err := runWhamCommand(t, "--config", configPath, "run", "all")

			// We expect an error during initialization.
			assert.Error(t, err, "The command should fail with an error exit code.")
			assert.Contains(t, outputStr, "Failed to initialize WHAM engine", "The error should be from the engine initialization.")
			assert.Contains(t, outputStr, "invalid configuration for step", "The error message should indicate an invalid step configuration.")
			assert.Contains(t, outputStr, tc.errContains, "The error message should contain the specific validation error.")
		})
	}
}

// TestInit_MergeConfigs verifies that loading multiple configuration files correctly
// merges them, with later files overriding earlier ones.
func TestInit_MergeConfigs(t *testing.T) {
	baseConfigPath := "../test/settings/settings_merge_base.yaml"
	overrideConfigPath := "../test/settings/settings_merge_override.yaml"
	goldenFilePath := "../test/golden/merged_config.json"

	// Load the configurations. This is the function under test.
	mergedConfig, err := cmd.LoadConfig(baseConfigPath, overrideConfigPath)
	assert.NoError(t, err, "Loading and merging configs should not produce an error.")

	// Marshal the resulting config object to JSON for comparison.
	resultJSON, err := json.MarshalIndent(mergedConfig, "", "  ")
	assert.NoError(t, err, "Marshalling merged config to JSON should not fail.")

	// Load the golden file as a template.
	goldenTemplateBytes, err := os.ReadFile(goldenFilePath)
	assert.NoError(t, err, "Should be able to read the golden file template.")

	// Parse and execute the template, injecting the dynamic ConfigDir.
	tmpl, err := template.New("golden").Parse(string(goldenTemplateBytes))
	assert.NoError(t, err, "Should be able to parse the golden file template.")

	var processedGolden bytes.Buffer
	templateContext := map[string]string{"ConfigDir": mergedConfig.ConfigDir}
	err = tmpl.Execute(&processedGolden, templateContext)
	assert.NoError(t, err, "Should be able to execute the golden file template.")

	// Compare the result with the golden file.
	assert.JSONEq(t, processedGolden.String(), string(resultJSON), "The merged config should match the golden file.")
}
