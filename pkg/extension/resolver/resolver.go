package resolver

import (
	"context"
)

const (
	PackageTypeFile    = "file"
	PackageTypeGithub  = "github"
	PackageTypeUnknown = "unknown"
)

type Resolver interface {
	// Resolve returns a binary path from a package reference
	Resolve(ctx context.Context, pkg string) (string, error)
}

// Source handles resolution for a specific scheme (e.g. github releases, local fs)
type Source interface {
	// Schema returns the URI scheme/prefix this source handles (e.g. "file", "github.com")
	Scheme() string

	// Resolve resolves a reference (without scheme prefix) to a binary path
	Resolve(ctx context.Context, ref string, opts ResolveOptions) (string, error)
}

type ResolveOptions struct {
	CacheDir string
	Platform string
}
