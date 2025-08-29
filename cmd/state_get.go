package cmd

import (
	"fmt"
	"os"
	"sort"
	"time"
)

// GetStepState displays the last known state of a single step.
func (w *WHAM) GetStepState(stepName string, outputFormat string) error {
	step := w.findStep(stepName)
	if step == nil {
		return fmt.Errorf("step '%s' not found", stepName)
	}

	state := w.getCurrentStepWhamState(stepName)

	switch outputFormat {
	case "json", "yaml":
		return RenderData(os.Stdout, state, outputFormat)
	case "table":
		// Reuse the 'all states' table renderer for consistency.
		return w.renderStatesAsTable([]Step{*step})
	default:
		return fmt.Errorf("unsupported output format: '%s'", outputFormat)
	}
}

// ShowExecutionSummary displays a summary table of the final state of all steps.
//
// It reads the last known state for each step from its corresponding WHAM state file
// and prints a formatted table with the step name, the last action performed
// ("run", "skipped", "failed"), the recorded run_id, and the timestamp of the run.
// Steps are sorted by DAG depth for readability.
func (w *WHAM) ShowExecutionSummary(outputFormat string) error {
	// Collect all states first, regardless of output format.
	switch outputFormat {
	case "json", "yaml":
		// For structured output, we collect states into a more descriptive struct.
		type namedState struct {
			StepName string `json:"step_name" yaml:"step_name"`
			StepState
		}
		var allNamedStates []namedState
		for _, step := range w.config.WhamSteps {
			state := w.getCurrentStepWhamState(step.Name)
			allNamedStates = append(allNamedStates, namedState{StepName: step.Name, StepState: state})
		}
		return RenderData(os.Stdout, allNamedStates, outputFormat)
	case "table":
		// For table output, we sort the steps first and then render them.
		stepsToSort := make([]Step, len(w.config.WhamSteps))
		copy(stepsToSort, w.config.WhamSteps)

		// Sort by depth for a consistent, logical order.
		sort.Slice(stepsToSort, func(i, j int) bool {
			depthI := w.stepDepths[stepsToSort[i].Name]
			depthJ := w.stepDepths[stepsToSort[j].Name]
			if depthI != depthJ {
				return depthI < depthJ
			}
			return stepsToSort[i].Name < stepsToSort[j].Name
		})
		return w.renderStatesAsTable(stepsToSort)
	default:
		return fmt.Errorf("unsupported output format: '%s'", outputFormat)
	}
}

func (w *WHAM) renderStatesAsTable(steps []Step) error {
	tr := NewTableRenderer(os.Stdout, "NAME", "ACTION", "RUN ID", "RUN DATE", "ELAPSED")

	for _, step := range steps {
		state := w.getCurrentStepWhamState(step.Name)
		runDate := "N/A"
		if !state.RunDate.IsZero() {
			runDate = state.RunDate.Format("2006-01-02 15:04:05")
		}
		elapsedStr := "N/A"
		if state.RunAction != "" { // Only show elapsed time if there's a state
			elapsedStr = state.Elapsed.Round(time.Millisecond).String()
		}
		tr.AddRow(step.Name, state.RunAction, state.RunID, runDate, elapsedStr)
	}

	return tr.Render()
}
