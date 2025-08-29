package cmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// whamBinaryPath holds the path to the compiled WHAM! executable for testing.
var whamBinaryPath string

// TestStepState is a struct used for unmarshaling the JSON output of `state get`.
// It mirrors the `namedState` struct used internally in the command.
type TestStepState struct {
	StepName  string        `json:"step_name"`
	RunAction string        `json:"run_action"`
	RunID     string        `json:"run_id,omitempty"`
	Elapsed   time.Duration `json:"elapsed,omitempty"`
}

// TestValidationResult is a struct used for unmarshaling the JSON output of `step validate`.
// It mirrors the `ValidationResult` struct used internally in the command.
type TestValidationResult struct {
	StepName string `json:"step_name"`
	Valid    bool   `json:"valid"`
	Reason   string `json:"reason"`
}

// TestDeletionResult is a struct used for unmarshaling the JSON output of `state delete`.
// It mirrors the `DeletionResult` struct used internally in the command.
type TestDeletionResult struct {
	StepName string `json:"step_name"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

// TestDAGStepInfo is a struct used for unmarshaling the JSON output of `dag get`.
// It mirrors the `DAGStepInfo` struct used internally in the command.
type TestDAGStepInfo struct {
	Name          string   `json:"name"`
	Depth         int      `json:"depth"`
	PreviousSteps []string `json:"previous_steps"`
}

// TestStep is a struct used for unmarshaling the JSON output of `step get`.
// It mirrors the `Step` struct from the `cmd` package.
type TestStep struct {
	Name string `json:"name"`
}

// TestMain is a special function that runs once for the entire test suite.
// It handles the setup (compiling the WHAM binary) and teardown (cleaning up
// the binary and test state files).
func TestMain(m *testing.M) {
	// --- SETUP ---
	var err error
	// Define the path for the temporary test binary.
	whamBinaryPath, err = filepath.Abs("./wham_test_binary")
	if err != nil {
		// If we can't even get an absolute path, we can't proceed.
		os.Exit(1)
	}

	// Compile the WHAM tool.
	buildCmd := exec.Command("go", "build", "-o", whamBinaryPath, "..")
	if err := buildCmd.Run(); err != nil {
		// If compilation fails, the tests cannot run.
		os.Exit(1)
	}

	// --- RUN TESTS ---
	// m.Run() executes all the test functions in this file.
	exitCode := m.Run()

	// --- TEARDOWN ---
	// Clean up the compiled binary.
	os.Remove(whamBinaryPath)
	// Do a final, broad cleanup of the default test state directories. Note the path change.
	cleanTestStates(nil, "../test/settings/settings_ok.yaml")

	os.Exit(exitCode)
}

// cleanTestStates removes the metadata and data directories used by the tests
// to ensure a clean slate for each test run.
// It now reads the config file to determine which directories to clean, making it more robust.
// It uses its own minimal config parsing to avoid circular dependencies with the `cmd` package.
func cleanTestStates(t *testing.T, configPath string) {
	// Minimal structs to unmarshal only what's needed for cleanup.
	type TestWhamSettings struct {
		DataDir     string `yaml:"data_dir"`
		MetadataDir string `yaml:"metadata_dir"`
	}
	type TestConfig struct {
		WhamSettings TestWhamSettings `yaml:"wham_settings"`
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if t != nil {
			t.Fatalf("Failed to load config '%s' for cleanup: %v", configPath, err)
		} else {
			println("Warning: Failed to read config for final cleanup:", err.Error())
		}
	}

	var config TestConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		if t != nil {
			t.Fatalf("Failed to parse YAML from '%s' for cleanup: %v", configPath, err)
		} else {
			println("Warning: Failed to parse YAML for final cleanup:", err.Error())
		}
	}
	// Helper function to clean the contents of a directory without deleting
	// the directory itself or dotfiles like .gitkeep.
	cleanDirContents := func(dir string) error {
		// Read all entries in the directory.
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil // Directory doesn't exist, so it's already "clean".
			}
			return err
		}
		for _, entry := range entries {
			// Explicitly skip the .gitkeep file to preserve it.
			if entry.Name() == ".gitkeep" {
				continue
			}
			// Remove all other files and subdirectories.
			if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	// Resolve paths relative to the config file's location, just like NewWHAM does.
	baseDir := filepath.Dir(configPath) // e.g., ../test/settings
	dataDir := config.WhamSettings.DataDir
	if !filepath.IsAbs(dataDir) {
		dataDir = filepath.Join(baseDir, dataDir)
	}

	metadataDir := config.WhamSettings.MetadataDir
	if !filepath.IsAbs(metadataDir) {
		metadataDir = filepath.Join(baseDir, metadataDir)
	}

	// --- SAFETY CHECKS ---
	// Prevent accidental deletion of the settings directory itself.
	// This can happen if data_dir or metadata_dir are empty or "." in the config.
	if dataDir == baseDir {
		if t != nil {
			t.Fatalf("SAFETY ABORT: data_dir in '%s' resolves to the settings directory itself ('%s'). Aborting cleanup.", configPath, dataDir)
		}
		panic("SAFETY ABORT: data_dir resolves to the settings directory itself. Aborting cleanup.")
	}
	if metadataDir == baseDir {
		if t != nil {
			t.Fatalf("SAFETY ABORT: metadata_dir in '%s' resolves to the settings directory itself ('%s'). Aborting cleanup.", configPath, metadataDir)
		}
		panic("SAFETY ABORT: metadata_dir resolves to the settings directory itself. Aborting cleanup.")
	}
	err1 := cleanDirContents(metadataDir)
	err2 := cleanDirContents(dataDir)

	if t != nil {
		// If called from a test, fail fast if cleanup has issues.
		assert.NoError(t, err1)
		assert.NoError(t, err2)
	}
}

// runWhamCommand is a helper function to execute the compiled WHAM! binary for tests.
// It centralizes command execution and sets the NO_COLOR environment variable
// to ensure that output is clean and free of ANSI color codes, making assertions reliable.
func runWhamCommand(t *testing.T, args ...string) (string, error) {
	cmd := exec.Command(whamBinaryPath, args...)
	// Set NO_COLOR to disable ANSI escape codes in the output from both
	// the WHAM logger and the test scripts.
	cmd.Env = append(os.Environ(), "NO_COLOR=true")

	// Use cmd.Output() to capture only stdout, which is essential for tests
	// that parse structured output like JSON. Stderr from logs will be printed
	// to the test runner's console, which is fine for debugging.
	stdout, err := cmd.Output()

	// If the command fails, the error (of type *exec.ExitError) will contain
	// the stderr. We combine stdout and stderr in the returned string for
	// more informative test failure messages.
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(stdout) + string(exitErr.Stderr), err
		}
	}
	return string(stdout), err
}
