package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// DAGStepInfo is a struct designed for structured output (JSON/YAML) of the DAG.
// It contains the essential information about a step's position in the graph.
type DAGStepInfo struct {
	Name          string   `json:"name" yaml:"name"`
	Depth         int      `json:"depth" yaml:"depth"`
	PreviousSteps []string `json:"previous_steps" yaml:"previous_steps"`
}

// GetDAG orchestrates the display of the workflow's Directed Acyclic Graph.
// It fetches the DAG structure and renders it in the format specified by `outputFormat`.
func (w *WHAM) GetDAG(outputFormat string) error {
	// The core logic to render the DAG is now in a separate function.
	// This function will handle the switch between different output formats.
	// For now, we'll keep the existing table rendering logic.
	return w.renderDAG(outputFormat)
}

// GetDAG displays the workflow's Directed Acyclic Graph to the console.
//
// The steps are rendered in a structured, human-readable format. They are sorted
// primarily by their calculated depth in the DAG and secondarily by name
// to ensure a stable and predictable output.
//
// To improve readability, the output is aligned: step names are padded to the same
// length, ensuring that the dependency arrows (`<--`) are vertically aligned.
func (w *WHAM) renderDAG(outputFormat string) error {
	// 1. Collect DAG information into a structured format.
	var dagInfo []DAGStepInfo
	for _, step := range w.config.WhamSteps {
		dagInfo = append(dagInfo, DAGStepInfo{
			Name:          step.Name,
			Depth:         w.stepDepths[step.Name],
			PreviousSteps: step.PreviousSteps,
		})
	}

	// Sort the collected info once, so all renderers use the same order.
	// Sort by depth (primary key) and name (secondary key, for stability).
	sort.Slice(dagInfo, func(i, j int) bool {
		if dagInfo[i].Depth != dagInfo[j].Depth {
			return dagInfo[i].Depth < dagInfo[j].Depth
		}
		return dagInfo[i].Name < dagInfo[j].Name
	})

	// 2. Render based on the requested format.
	switch outputFormat {
	case "json", "yaml":
		return RenderData(os.Stdout, dagInfo, outputFormat)
	case "table":
		return w.renderDAGAsTable(dagInfo)
	default:
		return fmt.Errorf("unsupported output format: '%s'", outputFormat)
	}
}

func (w *WHAM) renderDAGAsTable(dagInfo []DAGStepInfo) error {
	tr := NewTableRenderer(os.Stdout, "DEPTH", "NAME", "PREDECESSORS")

	for _, info := range dagInfo {
		depthStr := fmt.Sprintf("%d", info.Depth)

		predecessorsStr := "<none>"
		if len(info.PreviousSteps) > 0 {
			predecessorsStr = strings.Join(info.PreviousSteps, ", ")
		}

		tr.AddRow(depthStr, info.Name, predecessorsStr)
	}

	return tr.Render()
}
