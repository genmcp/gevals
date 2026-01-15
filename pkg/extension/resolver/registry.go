package resolver

import (
	"context"
	"fmt"
	"strings"
)

// Options configures the resolver behavior
type Options struct {
	// BasePath is the directory to use when resolving relative file paths
	BasePath string
}

// sources is a private package var holding all registered sources
var defaultSources = make(map[string]Source)

func init() {
	registerDefaultSource(&GithubSource{})
}

// registerDefaultSource registers a source to the default sources map
func registerDefaultSource(s Source) {
	scheme := s.Scheme()
	if _, ok := defaultSources[scheme]; ok {
		panic("only one resolver.Source can be registered for a given scheme, already had one registered for scheme " + scheme)
	}

	defaultSources[scheme] = s
}

// GetResolver creates a resolver with the given options
func GetResolver(opts Options) Resolver {
	sources := make(map[string]Source)

	// Copy default sources
	for k, v := range defaultSources {
		sources[k] = v
	}

	// Add FileSource with the configured base path
	sources[PackageTypeFile] = &FileSource{BasePath: opts.BasePath}

	return &registry{
		sources: sources,
	}
}

type registry struct {
	sources map[string]Source
}

var _ Resolver = &registry{}

func (r *registry) Resolve(ctx context.Context, pkg string) (string, error) {
	scheme, ref := parseRef(pkg)

	source, ok := r.sources[scheme]
	if !ok {
		return "", fmt.Errorf("unknown scheme in package reference %q", pkg)
	}

	return source.Resolve(ctx, ref)
}

func parseRef(ref string) (scheme, path string) {
	// file:// prefix
	if path, ok := strings.CutPrefix(ref, "file://"); ok {
		return PackageTypeFile, path
	}

	if strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "../") || strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "~/") {
		return PackageTypeFile, ref
	}

	if _, path, ok := strings.Cut(ref, "github.com/"); ok {
		return PackageTypeGithub, path
	}

	return PackageTypeUnknown, ref
}
