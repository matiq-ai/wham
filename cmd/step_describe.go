package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// DescribeStep prints the detailed configuration of a single step to the console.
//
// It retrieves the step's definition from the loaded configuration and displays
// all its properties in a human-readable format. This includes its script path,
// parameters, statefulness, dependencies, and environment variables.
//
// Returns an error if the specified step name is not found in the configuration.
func (w *WHAM) DescribeStep(stepName string) error {
	step := w.findStep(stepName)
	if step == nil {
		return fmt.Errorf("step '%s' not found", stepName)
	}

	// Use an errorWriter to simplify the printing logic.
	// We write to os.Stdout by default.
	ew := &errorWriter{w: os.Stdout}
	const keyFormat = "  %-18s: %s\n"

	ew.Printf("Name: %s\n", step.Name)

	// --- Configuration Section ---
	ew.Println("\nConfiguration:")
	ew.Printf(keyFormat, "Command", strings.Join(step.Command, " "))
	if step.Image != "" {
		ew.Printf(keyFormat, "Image", step.Image)
	}
	ew.Printf(keyFormat, "Args", formatStringSlice(step.Args))
	ew.Printf(keyFormat, "Stateful", fmt.Sprintf("%t", step.IsStateful))
	if step.WorkDir != "" {
		ew.Printf(keyFormat, "Work Dir", step.WorkDir)
	} else {
		ew.Printf(keyFormat, "Work Dir", "<default>")
	}
	if step.IsStateful {
		ew.Printf(keyFormat, "State File", step.StateFile)
		ew.Printf(keyFormat, "Run ID Var", step.RunIdVar)
	}
	ew.Printf(keyFormat, "Can Fail", fmt.Sprintf("%t", step.CanFail))
	ew.Printf(keyFormat, "Retries", fmt.Sprintf("%d", step.Retries))
	ew.Printf(keyFormat, "Retry Delay", step.RetryDelay.String())
	ew.Printf(keyFormat, "Previous Steps", formatPreviousSteps(step.PreviousSteps))

	ew.Println("  Env Vars:")
	if len(step.EnvVars) > 0 {
		// Sort keys for consistent output, which is good for testing and readability.
		keys := make([]string, 0, len(step.EnvVars))
		for k := range step.EnvVars {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			ew.Printf("    %s: %s\n", k, step.EnvVars[k])
		}
	} else {
		ew.Println("    <none>")
	}

	// --- State Section ---
	ew.Println("\nState:")
	state := w.getCurrentStepWhamState(stepName)
	if state.RunAction == "" {
		ew.Println("  <not run>")
	} else {
		runDate := "N/A"
		if !state.RunDate.IsZero() {
			runDate = state.RunDate.Format("2006-01-02 15:04:05")
		}
		ew.Printf(keyFormat, "Last Action", state.RunAction)
		ew.Printf(keyFormat, "Last Run ID", state.RunID)
		ew.Printf(keyFormat, "Last Run Date", runDate)
		ew.Printf(keyFormat, "Last Elapsed", state.Elapsed.Round(time.Millisecond).String())
	}

	// Return the first error that occurred, or nil if all writes succeeded.
	return ew.err
}

// DescribeAllSteps prints the detailed configuration for every step defined in the
// workflow.
//
// It iterates through the steps in the order they are defined in the configuration
// file (not the topological order) and calls `DescribeStep` for each one. A blank
// line is printed between each description for better readability.
//
// This function is useful for getting a complete overview of the entire workflow
// configuration at once.
func (w *WHAM) DescribeAllSteps() error {
	w.logger.Info().Msg("Describing all steps.")
	ew := &errorWriter{w: os.Stdout}
	// Iterate through the steps in the order they appear in the config file.
	for _, step := range w.config.WhamSteps {
		err := w.DescribeStep(step.Name)
		if err != nil {
			// This is unlikely to happen if the step exists in the config, but is handled for robustness.
			return err
		}
		ew.Println() // Add a blank line for better separation between step descriptions.
	}
	return ew.err
}

// formatStringSlice is a display helper for slices of strings.
func formatStringSlice(slice []string) string {
	if len(slice) == 0 {
		return "<none>"
	}
	return strings.Join(slice, " ")
}
