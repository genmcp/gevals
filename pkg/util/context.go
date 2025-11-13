package util

import (
	"context"
)

type contextKey string

const verboseKey contextKey = "verbose"

// WithVerbose adds the verbose flag to the context
func WithVerbose(ctx context.Context, verbose bool) context.Context {
	return context.WithValue(ctx, verboseKey, verbose)
}

// IsVerbose returns true if verbose mode is enabled in the context
func IsVerbose(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, ok := ctx.Value(verboseKey).(bool)
	return ok && v
}

