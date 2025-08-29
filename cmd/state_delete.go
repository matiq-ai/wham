package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// DeletionResult holds the outcome of a state deletion operation.
type DeletionResult struct {
	StepName string `json:"step_name" yaml:"step_name"`
	Status   string `json:"status" yaml:"status"`
	Message  string `json:"message" yaml:"message"`
}

// DeleteStepState orchestrates the deletion of one or all step states and renders the result.
func (w *WHAM) DeleteStepState(target string, outputFormat string, bypassPrompt bool) error {
	// Safety check: for any deletion, only proceed if the --yes flag is provided
	// or if the user confirms interactively.
	if !bypassPrompt {
		// Check if we are in an interactive terminal.
		if term.IsTerminal(int(os.Stdin.Fd())) {
			prompt := fmt.Sprintf("Are you sure you want to delete the state for '%s'? [y/N]: ", target)
			fmt.Print(prompt)
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(input)) != "y" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	var results []DeletionResult
	if target == "all" {
		for _, step := range w.config.WhamSteps {
			results = append(results, w.deleteSingleState(step.Name))
		}
	} else {
		// Ensure the step exists before trying to delete its state.
		if w.findStep(target) == nil {
			return fmt.Errorf("step '%s' not found", target)
		}
		results = []DeletionResult{w.deleteSingleState(target)}
	}

	switch outputFormat {
	case "json", "yaml":
		if len(results) == 1 {
			return RenderData(os.Stdout, results[0], outputFormat)
		}
		return RenderData(os.Stdout, results, outputFormat)
	case "table":
		return w.renderDeletionResultsAsTable(results)
	default:
		// This case is for future-proofing; kong should prevent invalid values.
		return fmt.Errorf("unsupported output format: '%s'", outputFormat)
	}
}

// deleteSingleState performs the actual file deletion for a step's state.
func (w *WHAM) deleteSingleState(stepName string) DeletionResult {
	stateFilePath := w.getWhamStateFilePath(stepName)
	err := os.Remove(stateFilePath)

	if err != nil {
		if os.IsNotExist(err) {
			w.logger.Info().Str("step", stepName).Msg("state file did not exist, already clean")
			return DeletionResult{StepName: stepName, Status: "already_clean", Message: "state file did not exist"}
		}
		// Handle other potential errors, like permissions.
		w.logger.Error().Str("step", stepName).Err(err).Msg("failed to delete state file")
		return DeletionResult{StepName: stepName, Status: "error", Message: err.Error()}
	}

	w.logger.Info().Str("step", stepName).Msg("state file deleted successfully")
	return DeletionResult{StepName: stepName, Status: "deleted", Message: "state file deleted successfully"}
}

// renderDeletionResultsAsTable displays deletion results in a table.
func (w *WHAM) renderDeletionResultsAsTable(results []DeletionResult) error {
	tr := NewTableRenderer(os.Stdout, "NAME", "STATUS", "MESSAGE")
	for _, res := range results {
		tr.AddRow(res.StepName, res.Status, res.Message)
	}
	return tr.Render()
}
