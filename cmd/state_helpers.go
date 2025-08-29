package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// getCurrentStepWhamState reads and parses the WHAM state file for a specific step.
//
// It constructs the path to the step's WHAM state file (e.g., wham_001_my-step.state)
// and attempts to read and unmarshal its JSON content into a StepState struct.
//
// If the file does not exist, cannot be read, or contains invalid JSON, the function
// logs the issue and returns an empty StepState{}. This is a safe default, as an
// empty run_id will typically trigger a re-run for dependent steps.
func (w *WHAM) getCurrentStepWhamState(stepName string) StepState {
	whamStateFilePath := w.getWhamStateFilePath(stepName)
	data, err := os.ReadFile(whamStateFilePath)
	if err != nil {
		// Handle cases where the file doesn't exist or can't be read.
		if os.IsNotExist(err) {
			w.logger.Debug().Str("step", stepName).Str("path", whamStateFilePath).Msg("WHAM state file does not exist, returning empty state.")
		} else {
			w.logger.Warn().Str("step", stepName).Str("path", whamStateFilePath).Err(err).Msg("Could not read WHAM state file, returning empty state.")
		}
		// Return an empty state, which is the expected behavior for a step that has never run.
		return StepState{}
	}

	var state StepState
	// The WHAM state files are stored in JSON format.
	err = json.Unmarshal(data, &state)
	if err != nil {
		w.logger.Warn().Str("step", stepName).Str("path", whamStateFilePath).Err(err).Msg("Could not parse WHAM state file, returning empty state.")
		// Return an empty state if the file is corrupted or not valid JSON.
		return StepState{}
	}
	return state
}

// saveStepWhamState creates and saves the WHAM state file for a specific step.
//
// It takes the step's name, its resulting run_id, and the action performed
// ("run", "skipped", or "failed"). It constructs a StepState object, marshals it
// into a human-readable JSON format, and writes it to the appropriate state file,
// overwriting any previous state. The file path is determined by getWhamStateFilePath.
//
// Returns an error if the JSON marshalling or file writing fails.
func (w *WHAM) saveStepWhamState(stepName, newRunID, action string, elapsed time.Duration) error {
	whamStateFilePath := w.getWhamStateFilePath(stepName)

	state := StepState{
		RunID:     newRunID,
		RunDate:   time.Now(),
		RunAction: action,
		Elapsed:   elapsed,
	}

	// Marshal the state to a human-readable, indented JSON format.
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal WHAM step state for '%s': %w", stepName, err)
	}

	// Write the state to the file with standard read/write permissions.
	err = os.WriteFile(whamStateFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write WHAM state file '%s': %w", whamStateFilePath, err)
	}

	w.logger.Debug().Str("step", stepName).Str("run_id", newRunID).Str("action", action).Str("path", whamStateFilePath).Msg("WHAM state saved.")
	return nil
}

// getWhamStateFilePath constructs the absolute path for a step's WHAM state file.
//
// The filename is assembled based on global settings.
//   - Base format: `[prefix][step_name][suffix]`
//   - With depth enabled (`metadata_add_depth: true`), the format becomes:
//     `[prefix][padded_depth]_[step_name][suffix]`
//
// The final path is created by joining this filename with the configured metadata directory.
// For example: `/path/to/metadata/wham_001_my-step.state`.
func (w *WHAM) getWhamStateFilePath(stepName string) string {
	// Default filename format without depth.
	filename := w.config.WhamSettings.MetadataPrefix + stepName + w.config.WhamSettings.MetadataSuffix

	// If configured, overwrite the filename to include the step's depth.
	if w.config.WhamSettings.MetadataAddDepth {
		depth := w.stepDepths[stepName]
		// Format the depth with leading zeros for consistent sorting (e.g., 001, 010, 100).
		depthStr := fmt.Sprintf("%0*d", w.config.WhamSettings.MetadataDepthPadding, depth)
		filename = w.config.WhamSettings.MetadataPrefix + depthStr + "_" + stepName + w.config.WhamSettings.MetadataSuffix
	}
	// Join with the absolute metadata directory path to get the full path.
	return filepath.Join(w.config.WhamSettings.MetadataDir, filename)
}
