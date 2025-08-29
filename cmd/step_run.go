package cmd

import (
	"fmt"
	"time"
)

// RunStep manages the execution of a single workflow step.
//
// It orchestrates the decision-making process (should the step run?), the
// execution itself, and the recording of the outcome in a WHAM state file.
//
// # Execution Logic
//
// The decision to execute a step is made as follows:
//  1. Forced Run: If `force` is true, the step is always executed, bypassing all checks.
//  2. Stateful Step: If the step `IsStateful` (and not forced), it is always executed.
//     The rationale is that a stateful step must run its internal logic to determine
//     if its state (and thus its `run_id`) has changed.
//  3. Stateless Step: If the step is stateless (and not forced), its execution depends
//     on the `shouldRunStep` helper. This function checks if the `run_id` of its
//     predecessors has changed since this step's last successful run. If a predecessor
//     is not ready (e.g., has not run yet), this function will return an error.
//
// # Outcome Recording
//
// After attempting to run, the outcome is saved to a WHAM state file:
//   - Success: The step's new `run_id` is determined. For a `stateful` step, the action
//     is always "run". For a `stateless` step, the action is "skipped" if its `run_id`
//     is unchanged, otherwise it is "run".
//   - Skipped (Pre-execution): If `shouldRunStep` returns false, the step is not executed.
//     The state is saved with the previous `run_id` and the action "skipped".
//   - Failure (`can_fail: true`): The script fails, but the workflow continues. The state
//     is saved with the action "failed". A `stateless` step inherits the `run_id` from
//     its predecessors to maintain DAG consistency, while a `stateful` step retains its
//     previous `run_id` as it failed to generate a new state.
//   - Failure (`can_fail: false`): The script fails, and the function returns an error,
//     halting the entire workflow.
func (w *WHAM) RunStep(stepName string, force bool) error {
	step := w.findStep(stepName)
	if step == nil {
		return fmt.Errorf("step '%s' not found", stepName)
	}

	w.logger.Debug().Str("step", stepName).Bool("force", force).Msg("Attempting to run step")

	// Pre-read current WHAM state (run_id from previous WHAM execution)
	prevWhamState := w.getCurrentStepWhamState(stepName)
	prevWhamRunID := prevWhamState.RunID // Can be empty if no previous state

	var shouldRun bool
	var elapsed time.Duration
	var err error

	if force {
		shouldRun = true // Always run if forced
		w.logger.Info().Str("step", stepName).Msg("Step forced to run.")
	} else if step.IsStateful {
		// Stateful steps are ALWAYS executed if not forced.
		// Their run_id is determined by their internal logic after execution.
		shouldRun = true
		w.logger.Info().Str("step", stepName).Msg("Stateful step will always execute (not forced).")
	} else { // Stateless step, not forced
		shouldRun, err = w.shouldRunStep(step)
		if err != nil {
			// An error from shouldRunStep indicates a precondition failure, such as
			// an inconsistent or not-yet-run predecessor.
			// The step is effectively skipped. We save this state and then return the
			// error to halt a `run all` workflow, ensuring the failure is propagated.
			w.saveStepWhamState(stepName, prevWhamRunID, "skipped", 0)
			fmt.Printf("🚫 Step '%s' skipped (precondition check failed).\n", stepName)
			w.logger.Warn().Str("step", stepName).Err(err).Msg("Step skipped due to precondition failure.")
			return fmt.Errorf("precondition check failed for step '%s': %w", stepName, err)
		}
	}

	if !shouldRun {
		// Stateless step skipped. Save WHAM state based on previous state.
		// A skipped step has an execution time of 0.
		w.saveStepWhamState(stepName, prevWhamRunID, "skipped", 0)
		fmt.Printf("✅ Step '%s' skipped (no changes detected).\n", stepName)
		w.logger.Info().Str("step", stepName).Msg("Stateless step skipped.")
		return nil
	}

	// --- Execute the step with retry logic ---
	var execErr error
	startTime := time.Now()
	// The loop runs for the initial attempt (attempt 0) plus the number of retries.
	for attempt := 0; attempt <= step.Retries; attempt++ {
		if attempt > 0 {
			w.logger.Warn().Str("step", step.Name).Int("attempt", attempt).Msgf("Retrying in %s...", step.RetryDelay)
			time.Sleep(step.RetryDelay)
		}
		fmt.Printf("🚀 Running step '%s' (attempt %d/%d)...\n", stepName, attempt+1, step.Retries+1)
		w.logger.Info().Str("step", stepName).Int("attempt", attempt+1).Int("total_attempts", step.Retries+1).Msg("Executing step.")

		execErr = w.executeStep(step, force, prevWhamRunID)
		if execErr == nil {
			break // Success, exit the retry loop
		}
	}

	// If execErr is not nil here, it means all attempts have failed.
	elapsed = time.Since(startTime)
	if execErr != nil {
		if step.CanFail {
			fmt.Printf("⚠️ Step '%s' failed but continuing (can_fail=true): %v\n", stepName, execErr)
			w.logger.Warn().Str("step", step.Name).Err(execErr).Msg("Step failed but allowed to continue.")
			// If a step with can_fail:true fails, we must decide which run_id to save.
			// - A STATELESS step inherits the run_id from its predecessors to maintain
			//   DAG consistency and avoid re-running unnecessarily.
			// - A STATEFUL step retains its previous run_id because it failed to generate
			//   a new internal state.
			// On failure, a step always retains its previous run_id. It did not successfully
			// complete the new run, so it should not adopt the new run_id. This preserves
			// an accurate history of the step's last known good state.
			runIdToSaveOnFailure := prevWhamRunID

			w.saveStepWhamState(step.Name, runIdToSaveOnFailure, "failed", elapsed)
		} else {
			w.logger.Error().Str("step", step.Name).Err(execErr).Msg("Step failed and cannot continue. Saving failed state.")
			// On a hard failure, we still save the state to record the failure event.
			// The run_id is the *previous* one, because the step did not successfully
			// complete a new run. If there was no previous run, this will be an empty string,
			// which correctly signals to dependent steps that this predecessor is not in a valid state.
			w.saveStepWhamState(step.Name, prevWhamRunID, "failed", elapsed)
			return fmt.Errorf("step '%s' failed: %w", stepName, execErr)
		}
	} else {
		// --- Step executed successfully, now update WHAM state ---
		// Get the run_id generated/updated by the script
		newActualRunID, err := w.getActualStepRunId(step)
		if err != nil {
			// The script ran successfully, but we can't determine its new state.
			// This is a critical failure that compromises the integrity of the DAG.
			return fmt.Errorf("step '%s' executed successfully, but failed to determine its new run_id: %w", step.Name, err)
		}
		w.logger.Debug().Str("step", step.Name).Str("new_actual_run_id", newActualRunID).Msg("New run ID from script execution.")

		// If execution reaches this point, the step was executed. The action is "run".
		// The "skipped" action is handled *before* the execution block based on shouldRunStep.
		runAction := "run"

		w.saveStepWhamState(step.Name, newActualRunID, runAction, elapsed)
		fmt.Printf("✅ Step '%s' completed successfully.\n", stepName)
		w.logger.Info().Str("step", step.Name).Msg("Step completed successfully.")
	}

	return nil
}

// RunAllSteps executes all defined steps in the workflow in their topological order.
//
// It first determines the correct execution sequence by calling `getTopologicalOrder`,
// which also validates the DAG for circular dependencies. It then iterates through the
// sorted steps, calling `RunStep` for each one.
//
// The `force` flag is passed down to each `RunStep` call, causing all steps to be
// executed unconditionally if set to true.
//
// If any step fails and is not marked with `can_fail: true`, the entire workflow
// is halted immediately, and the error from the failing step is returned.
func (w *WHAM) RunAllSteps(force bool, fromStep, toStep string) error {
	w.logger.Info().Bool("force", force).Str("from", fromStep).Str("to", toStep).Msg("Starting to run all steps.")

	// 1. Determine the correct execution order by performing a topological sort.
	// This also implicitly checks for circular dependencies in the DAG.
	sortedSteps, err := w.getTopologicalOrder()
	if err != nil {
		return fmt.Errorf("failed to determine step execution order: %w", err)
	}

	// 2. Filter the DAG based on --from and --to flags.
	stepsToRun, err := w.filterDAGForExecution(sortedSteps, fromStep, toStep)
	if err != nil {
		return err // An error here means an invalid --from/--to was provided.
	}

	// 3. Execute each step in the filtered and sorted list.
	for _, step := range stepsToRun {
		err := w.RunStep(step.Name, force)
		if err != nil {
			// If a step returns an error, it means it failed and did not have `can_fail: true`.
			// Halt the entire workflow immediately.
			w.logger.Error().Str("step", step.Name).Err(err).Msg("Workflow halted due to a failing step.")
			return err
		}
	}
	// If the loop completes, all steps have either succeeded, been skipped, or failed gracefully (with can_fail: true).
	w.logger.Info().Msg("All steps finished.")
	return nil
}

// filterDAGForExecution takes a topologically sorted list of all steps and filters it
// based on the --from and --to flags.
func (w *WHAM) filterDAGForExecution(allSteps []*Step, fromStepName, toStepName string) ([]*Step, error) {
	// If no flags are provided, run all steps.
	if fromStepName == "" && toStepName == "" {
		return allSteps, nil
	}

	// --- Build sets of valid steps for --from and --to ---
	runnableSteps := make(map[string]bool)

	// Handle --from: find all descendants of fromStepName.
	if fromStepName != "" {
		if w.findStep(fromStepName) == nil {
			return nil, fmt.Errorf("step specified in --from not found: '%s'", fromStepName)
		}
		descendants := make(map[string]bool)
		queue := []string{fromStepName}
		visited := make(map[string]bool)
		visited[fromStepName] = true
		descendants[fromStepName] = true

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			// Find direct successors of 'current'
			for _, step := range allSteps {
				for _, pred := range step.PreviousSteps {
					if pred == current && !visited[step.Name] {
						visited[step.Name] = true
						descendants[step.Name] = true
						queue = append(queue, step.Name)
					}
				}
			}
		}
		runnableSteps = descendants
	}

	// Handle --to: find all ancestors of toStepName.
	if toStepName != "" {
		if w.findStep(toStepName) == nil {
			return nil, fmt.Errorf("step specified in --to not found: '%s'", toStepName)
		}
		ancestors := make(map[string]bool)
		queue := []string{toStepName}
		visited := make(map[string]bool)
		visited[toStepName] = true
		ancestors[toStepName] = true

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, pred := range w.findStep(current).PreviousSteps {
				if !visited[pred] {
					visited[pred] = true
					ancestors[pred] = true
					queue = append(queue, pred)
				}
			}
		}

		// If --from was also specified, find the intersection.
		if fromStepName != "" {
			intersection := make(map[string]bool)
			for stepName := range runnableSteps {
				if ancestors[stepName] {
					intersection[stepName] = true
				}
			}
			runnableSteps = intersection
		} else {
			runnableSteps = ancestors
		}
	}

	// --- Filter the original sorted list to preserve order ---
	var finalStepsToRun []*Step
	for _, step := range allSteps {
		if runnableSteps[step.Name] {
			finalStepsToRun = append(finalStepsToRun, step)
		}
	}

	return finalStepsToRun, nil
}
