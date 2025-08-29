package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// GetStep orchestrates the display of one or all step configurations.
// It acts as a dispatcher, calling the appropriate function based on the target.
func (w *WHAM) GetStep(target string, outputFormat string) error {
	if target == "all" {
		return w.getAllSteps(outputFormat)
	}
	return w.getSingleStep(target, outputFormat)
}

// getSingleStep retrieves and displays the configuration for a single step.
func (w *WHAM) getSingleStep(stepName string, outputFormat string) error {
	step := w.findStep(stepName)
	if step == nil {
		return fmt.Errorf("step '%s' not found", stepName)
	}

	switch outputFormat {
	case "json", "yaml":
		return RenderData(os.Stdout, step, outputFormat)
	case "table":
		// Reuse the 'all steps' table renderer for consistency,
		// passing a slice with just the single step.
		return w.renderAllStepsAsTable([]Step{*step})
	default:
		return fmt.Errorf("unsupported output format: '%s'", outputFormat)
	}
}

// getAllSteps retrieves and displays the configuration for all steps.
func (w *WHAM) getAllSteps(outputFormat string) error {
	steps := w.config.WhamSteps

	switch outputFormat {
	case "json", "yaml":
		return RenderData(os.Stdout, steps, outputFormat)
	case "table":
		return w.renderAllStepsAsTable(steps)
	default:
		return fmt.Errorf("unsupported output format: '%s'", outputFormat)
	}
}

// renderAllStepsAsTable displays a summary of all steps in a table.
func (w *WHAM) renderAllStepsAsTable(steps []Step) error {
	tr := NewTableRenderer(os.Stdout, "NAME", "COMMAND", "STATEFUL", "CAN FAIL", "PREDECESSORS")

	for _, step := range steps {
		tr.AddRow(
			step.Name,
			strings.Join(step.Command, " "),
			strconv.FormatBool(step.IsStateful),
			strconv.FormatBool(step.CanFail),
			formatPreviousSteps(step.PreviousSteps),
		)
	}

	return tr.Render()
}
