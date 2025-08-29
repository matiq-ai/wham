package cmd

import (
	"fmt"
	"os"
	"strconv"
)

// ValidationResult holds the outcome of a step validation check.
type ValidationResult struct {
	StepName string `json:"step_name" yaml:"step_name"`
	Valid    bool   `json:"valid" yaml:"valid"`
	Reason   string `json:"reason" yaml:"reason"`
}

// GetValidationStatus orchestrates the validation of one or all steps and renders the result.
func (w *WHAM) GetValidationStatus(target string, outputFormat string) error {
	var results []ValidationResult
	var stepsToValidate []*Step

	if target == "all" {
		for i := range w.config.WhamSteps {
			stepsToValidate = append(stepsToValidate, &w.config.WhamSteps[i])
		}
	} else {
		step := w.findStep(target)
		if step == nil {
			// If the step is not found, treat it as a validation failure, not a fatal error.
			results = []ValidationResult{{StepName: target, Valid: false, Reason: "not found in configuration"}}
		} else {
			stepsToValidate = []*Step{step}
		}
	}

	// Only run validation if there are steps to validate.
	// This avoids running on a non-existent single target where `results` is already populated.
	if len(stepsToValidate) > 0 {
		results = w.validateSteps(stepsToValidate)
	}

	switch outputFormat {
	case "json", "yaml":
		// For a single step, output the object directly, not an array of one.
		if len(results) == 1 {
			return RenderData(os.Stdout, results[0], outputFormat)
		}
		return RenderData(os.Stdout, results, outputFormat)
	case "table":
		return w.renderValidationResultsAsTable(results)
	default:
		return fmt.Errorf("unsupported output format: '%s'", outputFormat)
	}
}

// validateSteps iterates through a slice of steps and collects their validation results.
func (w *WHAM) validateSteps(steps []*Step) []ValidationResult {
	var results []ValidationResult
	for _, step := range steps {
		_, err := w.validateStepExecutable(step)
		if err != nil {
			results = append(results, ValidationResult{StepName: step.Name, Valid: false, Reason: err.Error()})
		} else {
			results = append(results, ValidationResult{StepName: step.Name, Valid: true, Reason: "all checks ok"})
		}
	}
	return results
}

// renderValidationResultsAsTable displays validation results in a table.
func (w *WHAM) renderValidationResultsAsTable(results []ValidationResult) error {
	tr := NewTableRenderer(os.Stdout, "NAME", "VALID", "REASON")
	for _, res := range results {
		tr.AddRow(res.StepName, strconv.FormatBool(res.Valid), res.Reason)
	}
	return tr.Render()
}
