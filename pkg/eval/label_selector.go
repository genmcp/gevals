package eval

import (
	"fmt"
	"strings"
)

// ApplyLabelSelectorFilter applies a CLI-provided label selector (format: key=value)
// to an EvalSpec by merging it into each taskSet's LabelSelector (AND semantics).
//
// This is intentionally kept in the eval package so filtering logic is consolidated
// outside of the CLI layer.
func ApplyLabelSelectorFilter(spec *EvalSpec, selector string) error {
	if spec == nil {
		return fmt.Errorf("eval spec cannot be nil")
	}
	if selector == "" {
		return nil
	}

	// Parse label selector (format: key=value)
	parts := strings.SplitN(selector, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid label selector format, expected key=value, got: %s", selector)
	}
	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if key == "" || value == "" {
		return fmt.Errorf("label selector key and value cannot be empty")
	}

	// Filter taskSets that match the label selector
	var filteredTaskSets []TaskSet
	for _, ts := range spec.Config.TaskSets {
		// Merge CLI selector into taskSet selector (AND semantics)
		if ts.LabelSelector == nil {
			ts.LabelSelector = make(map[string]string)
		}
		if existing, exists := ts.LabelSelector[key]; exists && existing != value {
			continue // incompatible selector
		}
		ts.LabelSelector[key] = value
		filteredTaskSets = append(filteredTaskSets, ts)
	}

	if len(filteredTaskSets) == 0 {
		return fmt.Errorf("no taskSets match label selector %s=%s", key, value)
	}

	// Replace taskSets with filtered ones
	spec.Config.TaskSets = filteredTaskSets

	return nil
}

// matchesLabelSelector checks if the task labels match the label selector.
// All labels in the selector must match (AND logic).
// Returns true if selector is empty or nil.
func matchesLabelSelector(taskLabels, selector map[string]string) bool {
	if len(selector) == 0 {
		return true
	}

	for key, value := range selector {
		taskValue, exists := taskLabels[key]
		if !exists || taskValue != value {
			return false
		}
	}

	return true
}
