package agent

import "strings"

// normalizeModelName converts model names to valid env var format
// Example: "gemini-2.5-pro" -> "GEMINI_2_5_PRO"
func normalizeModelName(name string) string {
	normalized := strings.ToUpper(name)
	normalized = strings.ReplaceAll(normalized, "-", "_")
	normalized = strings.ReplaceAll(normalized, ".", "_")
	return normalized
}
