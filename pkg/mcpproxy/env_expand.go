package mcpproxy

import (
	"fmt"
	"os"
	"regexp"
)

var (
	// Pattern for ${VAR:-default} syntax (must come before ${VAR} pattern)
	envWithDefaultPattern = regexp.MustCompile(`\$\{([^:}]+):-([^}]*)\}`)
	// Pattern for ${VAR} syntax (required) - matches only if not already matched by default pattern
	envRequiredPattern = regexp.MustCompile(`\$\{([^}]+)\}`)
)

// ExpandEnv expands environment variable references in a string value.
// Supports:
//   - ${VAR} - required variable (error if not set)
//   - ${VAR:-default} - optional with default value
//
// Returns the expanded string or an error if a required variable is missing.
// Nested expansions are supported and processed recursively.
func ExpandEnv(value string) (string, error) {
	result := value
	maxIterations := 10 // Prevent infinite loops
	iteration := 0

	for iteration < maxIterations {
		iteration++
		prevResult := result

		// First, expand ${VAR:-default} patterns (these take precedence)
		result = envWithDefaultPattern.ReplaceAllStringFunc(result, func(match string) string {
			submatches := envWithDefaultPattern.FindStringSubmatch(match)
			if len(submatches) != 3 {
				return match
			}
			varName := submatches[1]
			defaultValue := submatches[2]

			if val, ok := os.LookupEnv(varName); ok && val != "" {
				return val
			}
			// Expand the default value recursively
			expandedDefault, err := expandOnce(defaultValue)
			if err == nil && expandedDefault != defaultValue {
				return expandedDefault
			}
			return defaultValue
		})

		// Then, expand ${VAR} patterns (required) - but skip any that contain :- (already processed)
		missingVars := []string{}
		result = envRequiredPattern.ReplaceAllStringFunc(result, func(match string) string {
			// Skip if this contains :- (already handled by default pattern)
			if regexp.MustCompile(`:-`).MatchString(match) {
				return match
			}

			submatches := envRequiredPattern.FindStringSubmatch(match)
			if len(submatches) != 2 {
				return match
			}
			varName := submatches[1]

			val, ok := os.LookupEnv(varName)
			if !ok || val == "" {
				missingVars = append(missingVars, match)
				return match
			}
			return val
		})

		// Check if any required variables are missing
		if len(missingVars) > 0 {
			return "", fmt.Errorf("required environment variable(s) not set: %v", missingVars)
		}

		// If nothing changed, we're done
		if result == prevResult {
			break
		}
	}

	return result, nil
}

// expandOnce performs a single pass of expansion (used for nested defaults)
func expandOnce(value string) (string, error) {
	result := value

	// Expand ${VAR:-default} patterns
	result = envWithDefaultPattern.ReplaceAllStringFunc(result, func(match string) string {
		submatches := envWithDefaultPattern.FindStringSubmatch(match)
		if len(submatches) != 3 {
			return match
		}
		varName := submatches[1]
		defaultValue := submatches[2]

		if val, ok := os.LookupEnv(varName); ok && val != "" {
			return val
		}
		return defaultValue
	})

	// Expand ${VAR} patterns (but skip if they contain :-)
	missingVars := []string{}
	result = envRequiredPattern.ReplaceAllStringFunc(result, func(match string) string {
		if regexp.MustCompile(`:-`).MatchString(match) {
			return match
		}

		submatches := envRequiredPattern.FindStringSubmatch(match)
		if len(submatches) != 2 {
			return match
		}
		varName := submatches[1]

		val, ok := os.LookupEnv(varName)
		if !ok || val == "" {
			missingVars = append(missingVars, match)
			return match
		}
		return val
	})

	if len(missingVars) > 0 {
		return "", fmt.Errorf("required environment variable(s) not set: %v", missingVars)
	}

	return result, nil
}

