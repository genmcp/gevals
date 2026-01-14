package resolver

import (
	"context"
	"fmt"
	"strings"
)

// sources is a private package var holding all registered sources
var sources = make(map[string]Source)

func init() {
	RegisterSourceOrDie(&GithubSource{})
	RegisterSourceOrDie(&FileSource{})
}

// RegisterSourceOrDie registers a source, or panics if a source for that scheme is already registered
func RegisterSourceOrDie(s Source) {
	scheme := s.Scheme()
	if _, ok := sources[scheme]; ok {
		panic("only one resolver.Source can be registered for a given scheme, already had one registered for scheme " + scheme)
	}

	sources[scheme] = s
}

func GetResolver() Resolver {
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
