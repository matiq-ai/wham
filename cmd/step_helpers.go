package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// TemplateContext holds dynamic data available at runtime for a step's execution.
// This data is passed to the template engine when processing parameter strings.
type TemplateContext struct {
	Forced   bool             // True if the step was forced to run.
	Step     *Step            // A pointer to the step's own configuration.
	RunID    string           // The step's run_id from its previous execution.
	Config   *Config          // A pointer to the entire WHAM configuration.
	StepsMap map[string]*Step // A map of all steps for easy lookup by name.
}

// Helper methods

// findStep retrieves a pointer to a Step definition by its name.
// It performs a fast lookup using an internal map for efficiency.
// Returns nil if no step with the given name is found.
func (w *WHAM) findStep(name string) *Step {
	return w.stepsMap[name]
}

// shouldRunStep determines if a stateless step, in a non-forced run, should be executed.
//
// This function is the core of the conditional execution logic for stateless steps.
// It compares the current combined `run_id` of the step's predecessors with the
// `run_id` recorded during this step's last execution.
//
// Logic:
//  1. If the step has predecessors, it fetches their `run_id`s. If they are consistent
//     (all the same and not empty), it compares this common `run_id` with the step's
//     own last known `run_id`. It returns `true` if they differ, `false` otherwise.
//  2. If the step has no predecessors (it's a source node), it always returns `true`
//     as there is no prior state to compare against.
//  3. It returns an error if any predecessor is not ready (missing a state file or `run_id`)
//     or if predecessors have inconsistent `run_id`s.
func (w *WHAM) shouldRunStep(step *Step) (bool, error) {
	// Get the run_id from this step's last execution.
	currentWhamRunID := w.getCurrentStepWhamState(step.Name).RunID
	w.logger.Debug().Str("step", step.Name).Str("current_wham_run_id", currentWhamRunID).Msg("Current WHAM run ID for stateless step.")

	if len(step.PreviousSteps) > 0 {
		// Get the consistent run_id from all direct predecessors.
		// This will return an error if any predecessor is not ready or if they are inconsistent.
		prevRunID, err := w.checkPreviousStepsConsistency(step.PreviousSteps)
		if err != nil {
			return false, err // Propagate the error to halt execution.
		}
		w.logger.Debug().Str("step", step.Name).Str("previous_steps_consistent_run_id", prevRunID).Msg("Consistent run ID from previous steps for stateless step.")

		// If the consistent run_id from predecessors is empty, it implies that all
		// predecessors were of a type that doesn't contribute a run_id (e.g.,
		// stateless source nodes or can_fail steps). In this scenario, the current
		// step should always run, as there's no meaningful prior state to compare against.
		if prevRunID == "" {
			return true, nil
		}
		// Run only if the predecessors' state has changed since our last run.
		return prevRunID != currentWhamRunID, nil
	}

	// A stateless step with no predecessors should always run.
	return true, nil
}

// checkPreviousStepsConsistency verifies that all direct predecessors of a step are in a
// consistent and ready state.
//
// It iterates through the list of predecessor names and performs two critical checks:
//  1. Readiness: Each predecessor must have a valid WHAM state with a non-empty `run_id`.
//     An empty `run_id` implies the predecessor has not run successfully yet.
//  2. Consistency: All predecessors must have the *exact same* `run_id`.
//
// If any check fails, it returns an error to prevent the dependent step from running.
// If all checks pass, it returns the common `run_id` shared by all predecessors.
func (w *WHAM) checkPreviousStepsConsistency(previousSteps []string) (string, error) {
	var commonRunID string
	var firstStepChecked string

	for _, stepName := range previousSteps {
		predStep := w.findStep(stepName) // Get the predecessor's definition

		// Case 1: Handle stateless source nodes.
		// It's acceptable for them to have no run_id, as they are just entry points.
		// We can safely skip them in consistency checks.
		if predStep != nil && !predStep.IsStateful && len(predStep.PreviousSteps) == 0 {
			w.logger.Debug().Str("previous_step", stepName).Msg("Skipping run_id consistency check for stateless source node.")
			continue
		}

		whamState := w.getCurrentStepWhamState(stepName)
		w.logger.Debug().Str("previous_step", stepName).Str("wham_run_id", whamState.RunID).Msg("Checking previous step WHAM run ID.")

		// Case 2: If a predecessor can fail, we accept its state as-is (potentially stale)
		// and skip the consistency check for it. We only care that it has run at least once.
		if predStep != nil && predStep.CanFail {
			w.logger.Warn().Str("previous_step", stepName).Str("stale_run_id", whamState.RunID).Msg("Accepting potentially stale state from predecessor marked with 'can_fail'.")
			continue
		}

		// Case 3: Hard failure for any other step without a run_id.
		// This means the step has never completed successfully, and we cannot proceed.
		// This check happens *after* the can_fail check.
		if whamState.RunID == "" {
			return "", fmt.Errorf("previous step '%s' has no valid WHAM state (empty run_id). Cannot proceed with dependent step", stepName)
		}

		// Case 4: Establish the reference run_id from the first valid predecessor.
		if commonRunID == "" {
			commonRunID = whamState.RunID
			firstStepChecked = stepName
		} else if commonRunID != whamState.RunID {
			// This is a critical inconsistency for a step that cannot fail. Halt the workflow.
			return "", fmt.Errorf("previous steps have inconsistent run_ids: '%s' has '%s', but '%s' has '%s'",
				firstStepChecked, commonRunID, stepName, whamState.RunID)
		}
	}

	return commonRunID, nil
}

// getActualStepRunId determines the definitive run_id for a step *after* its execution
// and returns it.
//
// The method of determination depends on whether the step is stateful or stateless:
//   - For a `stateful` step, it reads the `state_file` (e.g., `my_state.state`)
//     generated by the script in the metadata directory. It then parses this file
//     to find the line containing the configured `run_id_var` (e.g., `run_id=some_value`)
//     and extracts the value. It returns an empty string with no error if the file is
//     missing, unreadable, or the `run_id_var` is not found.
//   - For a `stateless` step, it inherits the consistent `run_id` from its direct
//     predecessors. If predecessors are inconsistent, it returns an error. If it has
//     no predecessors, it returns an empty string.
func (w *WHAM) getActualStepRunId(step *Step) (string, error) {
	if step.IsStateful {
		// For stateful steps, the run_id is read from the state file they generate.
		stepStateFilePath := filepath.Join(w.config.WhamSettings.MetadataDir, step.StateFile)

		data, err := os.ReadFile(stepStateFilePath)
		if err != nil {
			if os.IsNotExist(err) {
				w.logger.Warn().Str("step", step.Name).Str("path", stepStateFilePath).Msg("Stateful step's state file does not exist after execution. Using empty string as run_id.")
			} else {
				w.logger.Error().Str("step", step.Name).Str("path", stepStateFilePath).Err(err).Msg("Failed to read stateful step's state file after execution.")
			}
			return "", nil // If the file doesn't exist or can't be read, there's no valid run_id.
		}

		// Parse the file content line-by-line to find the run_id variable (e.g., "run_id=...").
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, step.RunIdVar+"=") {
				runID := strings.TrimPrefix(line, step.RunIdVar+"=")
				return strings.TrimSpace(runID), nil
			}
		}

		// If the run_id_var line is not found in the file.
		w.logger.Warn().Str("step", step.Name).Str("path", stepStateFilePath).Str("run_id_var", step.RunIdVar).Msg("Run ID variable not found in stateful step's state file.")
		return "", nil
	}
	// For stateless steps, the run_id is derived from its predecessors.
	if len(step.PreviousSteps) == 0 {
		// A stateless source node has no predecessors to inherit from. Its run_id is empty.
		return "", nil
	}
	// Inherit the run_id from predecessors. This call also validates their consistency.
	prevRunID, err := w.checkPreviousStepsConsistency(step.PreviousSteps)
	if err != nil {
		// If we can't get a consistent run_id (e.g., a predecessor hasn't run),
		// the resulting run_id for this step is effectively empty. This can happen
		// during a forced run of a single step. We log it but don't treat it as a fatal error.
		w.logger.Warn().Str("step", step.Name).Err(err).Msg("Could not determine a consistent run_id from predecessors. Defaulting to empty string.")
		return "", nil
	}
	return prevRunID, nil
}

// executeStep handles the actual execution of an external script defined by a Step.
//
// This function orchestrates several key tasks:
//  1. Path Resolution: It resolves the script path to an absolute path, using the
//     configuration file's directory as a base for relative paths.
//  2. Pre-flight Checks: It performs a quick check to ensure the script file exists,
//     is not a directory, and has execute permissions before attempting to run it.
//  3. Argument Assembly: It combines any shared parameters from `wham_settings` with
//     the step-specific parameters.
//  4. Environment Setup: It prepares the environment for the script by:
//     - Inheriting the parent process's environment.
//     - Injecting WHAM-specific variables (`VAR_DATA_DIR`, `VAR_METADATA_DIR`).
//     - Adding any custom environment variables defined for the step.
//  5. Execution: It runs the command and pipes the script's stdout and stderr to the
//     main WHAM process to ensure visibility of its output.
//
// Returns an error if any part of the setup or the script execution itself fails.
func (w *WHAM) executeStep(step *Step, force bool, prevRunID string) error {
	executable, err := w.validateStepExecutable(step)
	if err != nil {
		return err // Error already contains context about the step name.
	}

	// 3. Assemble command-line arguments with runtime templating.
	templateContext := TemplateContext{
		Forced:   force,      // Is this a forced run?
		Step:     step,       // The current step's data.
		RunID:    prevRunID,  // The previous run_id for this step.
		Config:   w.config,   // The entire configuration.
		StepsMap: w.stepsMap, // Provide access to all steps by name.
	}

	// Combine command, shared, and local args into the final args slice.
	// Start with the arguments from the command definition itself.
	args := step.Command[1:]

	// Process and append shared args. Each template can expand into multiple space-separated arguments.
	for _, sharedArgTpl := range w.config.WhamSettings.SharedArgs {
		processedArg, err := w.processTemplateString(sharedArgTpl, templateContext)
		if err != nil {
			return fmt.Errorf("failed to process shared_arg template '%s' for step '%s': %w", sharedArgTpl, step.Name, err)
		}
		if processedArg != "" {
			args = append(args, strings.Fields(processedArg)...)
		}
	}

	// Process and append local args. Each element in the slice is a single argument.
	for _, argTpl := range step.Args {
		processedArg, err := w.processTemplateString(argTpl, templateContext)
		if err != nil {
			return fmt.Errorf("failed to process arg template '%s' for step '%s': %w", argTpl, step.Name, err)
		}
		// Append the processed argument as a whole. This handles spaces correctly.
		if processedArg != "" {
			args = append(args, processedArg)
		}
	}

	// 4. Prepare the command and its environment.
	cmd := exec.Command(executable, args...)
	cmd.Env = os.Environ() // Inherit the current process's environment.

	// Set the working directory for the script if specified.
	if step.WorkDir != "" {
		workDir := step.WorkDir
		// Resolve relative paths based on the config file's directory.
		if !filepath.IsAbs(workDir) {
			workDir = filepath.Join(w.config.ConfigDir, workDir)
		}
		workDir = filepath.Clean(workDir)

		// Verify the working directory exists and is a directory.
		stat, err := os.Stat(workDir)
		if err != nil || !stat.IsDir() {
			return fmt.Errorf("invalid work_dir '%s' for step '%s': path does not exist or is not a directory", step.WorkDir, step.Name)
		}
		cmd.Dir = workDir
	}

	cmd.Env = append(cmd.Env, fmt.Sprintf("VAR_DATA_DIR=%s", w.config.WhamSettings.DataDir))
	cmd.Env = append(cmd.Env, fmt.Sprintf("VAR_METADATA_DIR=%s", w.config.WhamSettings.MetadataDir))
	for k, v := range step.EnvVars {
		// Process the template for the value of the environment variable.
		processedVal, err := w.processTemplateString(v, templateContext)
		if err != nil {
			// Provide a more specific error message.
			return fmt.Errorf("failed to process template for env_var '%s' in step '%s': %w", k, step.Name, err)
		}
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, processedVal))
	}

	// 5. Execute the command and stream its output.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	w.logger.Debug().Str("step", step.Name).Str("command", cmd.String()).Interface("templateContext", templateContext).Msg("Executing command with runtime context.")

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("script execution failed: %w", err)
	}

	return nil
}

// validateStepExecutable centralizes the logic for checking if a step's command is valid.
// It checks for existence, ensures it's a file (not a directory), and verifies execute permissions.
// It returns the absolute, cleaned path to the executable on success.
func (w *WHAM) validateStepExecutable(step *Step) (string, error) {
	// 1. Validate and resolve the command executable.
	if len(step.Command) == 0 {
		return "", fmt.Errorf("step '%s' has an empty 'command' definition", step.Name)
	}
	executable := step.Command[0]
	if !filepath.IsAbs(executable) {
		executable = filepath.Join(w.config.ConfigDir, executable)
	}
	executable = filepath.Clean(executable) // Normalize path.

	// 2. Perform file system checks on the executable file.
	stat, err := os.Stat(executable)
	if err != nil {
		return "", fmt.Errorf("command executable '%s' for step '%s' not found", executable, step.Name)
	}
	if stat.IsDir() {
		return "", fmt.Errorf("command path '%s' for step '%s' is a directory", executable, step.Name)
	}
	// Check if any of the executable bits (owner, group, or other) are set.
	if stat.Mode()&0111 == 0 {
		return "", fmt.Errorf("command executable '%s' for step '%s' is not executable", executable, step.Name)
	}

	return executable, nil
}

// formatPreviousSteps is a display helper that formats a slice of predecessor names
// into a human-readable string.
//
// If the slice is empty, it returns the string "none". Otherwise, it returns a
// comma-separated list of the step names.
func formatPreviousSteps(steps []string) string {
	if len(steps) == 0 {
		return "<none>"
	}
	return strings.Join(steps, ", ")
}
