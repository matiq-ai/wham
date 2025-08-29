package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

// TestConfigGet_MergedOutput verifies that `config get` correctly displays the
// final configuration after merging multiple files, matching the golden file.
func TestConfigGet_MergedOutput(t *testing.T) {
	baseConfigPath := "../test/settings/settings_merge_base.yaml"
	overrideConfigPath := "../test/settings/settings_merge_override.yaml"
	goldenFilePath := "../test/golden/merged_config.json"

	// Run the 'config get' command with JSON output.
	outputStr, err := runWhamCommand(t, "--config", baseConfigPath, "--config", overrideConfigPath, "config", "get", "-o", "json")
	assert.NoError(t, err, "config get -o json should succeed.")

	// To correctly process the golden file template, we need the absolute path
	// of the base config directory, just like the main application does.
	configDir, err := filepath.Abs(filepath.Dir(baseConfigPath))
	assert.NoError(t, err, "Should be able to get the absolute path of the config directory.")

	// Load the golden file as a template.
	goldenTemplateBytes, err := os.ReadFile(goldenFilePath)
	assert.NoError(t, err, "Should be able to read the golden file template.")

	// Parse and execute the template, injecting the dynamic ConfigDir.
	tmpl, err := template.New("golden").Parse(string(goldenTemplateBytes))
	assert.NoError(t, err, "Should be able to parse the golden file template.")

	var processedGolden bytes.Buffer
	templateContext := map[string]string{"ConfigDir": configDir}
	err = tmpl.Execute(&processedGolden, templateContext)
	assert.NoError(t, err, "Should be able to execute the golden file template.")

	// Compare the command's JSON output with the processed golden file.
	assert.JSONEq(t, processedGolden.String(), outputStr, "The output of 'config get' should match the golden file.")
}
