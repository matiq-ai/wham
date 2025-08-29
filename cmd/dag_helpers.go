package cmd

import "fmt"

// getTopologicalOrder performs a topological sort of the workflow's Directed Acyclic Graph (DAG).
//
// This function is crucial for determining the correct execution order of steps and for
// detecting circular dependencies, which would otherwise cause an infinite loop.
//
// It implements Kahn's algorithm, which works as follows:
//  1. Compute In-degrees: It calculates the number of incoming dependencies for each step.
//     At the same time, it builds an adjacency list to represent the graph and validates
//     that all declared predecessors exist.
//  2. Initialize Queue: It identifies all "source nodes" (steps with an in-degree of 0)
//     and adds them to a queue.
//  3. Process Nodes: It dequeues steps one by one, adding them to the sorted list. For
//     each dequeued step, it "removes" its outgoing edges by decrementing the in-degree
//     of its successors. If a successor's in-degree becomes 0, it's added to the queue.
//  4. Detect Cycles: After the loop, if the number of steps in the sorted list is less
//     than the total number of steps, the graph contains a cycle, and an error is returned.
func (w *WHAM) getTopologicalOrder() ([]*Step, error) {
	// Step 1: Compute in-degrees and build the adjacency list (successors map).
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	for _, step := range w.config.WhamSteps {
		inDegree[step.Name] = len(step.PreviousSteps)
		for _, prevStepName := range step.PreviousSteps {
			// Validate that the declared predecessor actually exists in the configuration.
			if _, ok := w.stepsMap[prevStepName]; !ok {
				return nil, fmt.Errorf("step '%s' declares non-existent previous step '%s'", step.Name, prevStepName)
			}
			// An edge from prevStepName to step.Name means step.Name is a successor of prevStepName.
			adjList[prevStepName] = append(adjList[prevStepName], step.Name)
		}
	}

	// Step 2: Initialize a queue with all nodes having an in-degree of 0 (source nodes).
	var queue []string
	for _, step := range w.config.WhamSteps {
		if inDegree[step.Name] == 0 {
			queue = append(queue, step.Name)
		}
	}

	// Step 3: Process the queue to build the sorted list.
	var sortedSteps []*Step
	for len(queue) > 0 {
		currentStepName := queue[0]
		queue = queue[1:]
		sortedSteps = append(sortedSteps, w.stepsMap[currentStepName])

		// For each successor of the current step, decrement its in-degree.
		for _, neighborStepName := range adjList[currentStepName] {
			inDegree[neighborStepName]--
			// If a successor's in-degree drops to 0, it becomes a new source node.
			if inDegree[neighborStepName] == 0 {
				queue = append(queue, neighborStepName)
			}
		}
	}

	// Step 4: Check for cycles.
	if len(sortedSteps) != len(w.config.WhamSteps) {
		return nil, fmt.Errorf("circular dependency detected in workflow DAG")
	}

	return sortedSteps, nil
}

func (w *WHAM) calculateStepDepths() {
	// 1. Get the topological order. This also validates the DAG for cycles.
	sortedSteps, err := w.getTopologicalOrder()
	if err != nil {
		// If a topological sort is not possible (e.g., due to a cycle), we cannot calculate depths.
		// Log the error and set all depths to 0 as a safe fallback.
		w.logger.Error().Err(err).Msg("Could not determine topological order for depth calculation. Defaulting all depths to 0.")
		for _, step := range w.config.WhamSteps {
			w.stepDepths[step.Name] = 0
		}
		return
	}

	// 2. Initialize all depths to 0.
	for _, step := range w.config.WhamSteps {
		w.stepDepths[step.Name] = 0
	}

	// 3. Build an adjacency list to easily find the successors of each node.
	//    key: predecessor name, value: list of successor names
	adjList := make(map[string][]string)
	for _, step := range w.config.WhamSteps {
		for _, prevStepName := range step.PreviousSteps {
			adjList[prevStepName] = append(adjList[prevStepName], step.Name)
		}
	}

	// 4. Iterate through the topologically sorted steps to calculate depths.
	for _, u := range sortedSteps { // 'u' is the current step
		for _, vName := range adjList[u.Name] { // 'vName' is the name of a successor of 'u'
			// The new potential depth for the successor is the current node's depth + 1.
			if newDepth := w.stepDepths[u.Name] + 1; newDepth > w.stepDepths[vName] {
				w.stepDepths[vName] = newDepth
			}
		}
	}
}
