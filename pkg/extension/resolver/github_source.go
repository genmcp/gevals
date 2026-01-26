package resolver

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/genmcp/gen-mcp/pkg/utils/binarycache"
)

// GithubSource resolves extension binaries from GitHub releases
type GithubSource struct{}

var _ Source = &GithubSource{}

func (s *GithubSource) Scheme() string {
	return PackageTypeGithub
}

// Resolve resolves a GitHub reference to a local binary path.
// The ref format is: owner/repo[@version]
// Examples:
//   - "myorg/myext" - resolves to latest version
//   - "myorg/myext@v1.0.0" - resolves to specific version
func (s *GithubSource) Resolve(ctx context.Context, ref string) (string, error) {
	owner, repo, version, err := parseGithubRef(ref)
	if err != nil {
		return "", err
	}

	cfg := &binarycache.Config{
		CacheName:              fmt.Sprintf(".mcpchecker/%s-%s", owner, repo),
		BinaryPrefix:           repo,
		GitHubReleasesURL:      fmt.Sprintf("https://github.com/%s/%s/releases/download", owner, repo),
		GitHubAPIURL:           fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo),
		SigstoreIdentityRegexp: fmt.Sprintf("https://github.com/%s/%s/.*", owner, repo),
		SigstoreOIDCIssuer:     "https://token.actions.githubusercontent.com",
	}

	downloader, err := binarycache.NewBinaryDownloader(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create binary downloader: %w", err)
	}

	binaryPath, err := downloader.GetBinary(version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", fmt.Errorf("failed to get binary for %s/%s@%s: %w", owner, repo, version, err)
	}

	return binaryPath, nil
}

// parseGithubRef parses a GitHub reference into owner, repo, and version
// Format: owner/repo[@version]
func parseGithubRef(ref string) (owner, repo, version string, err error) {
	// Split off version if present
	refPart := ref
	version = "latest"

	if idx := strings.LastIndex(ref, "@"); idx != -1 {
		refPart = ref[:idx]
		version = ref[idx+1:]
		if version == "" {
			return "", "", "", fmt.Errorf("invalid github reference '%s': empty version after @", ref)
		}
	}

	// Parse owner/repo
	parts := strings.Split(refPart, "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid github reference '%s': expected format owner/repo[@version]", ref)
	}

	owner = parts[0]
	repo = parts[1]

	if owner == "" || repo == "" {
		return "", "", "", fmt.Errorf("invalid github reference '%s': owner and repo cannot be empty", ref)
	}

	return owner, repo, version, nil
}

