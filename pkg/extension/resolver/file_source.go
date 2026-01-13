package resolver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const isExecutableMask = 0111

type FileSource struct{}

var _ Source = &FileSource{}

func (s *FileSource) Scheme() string {
	return PackageTypeFile
}

func (s *FileSource) Resolve(ctx context.Context, ref string, opts ResolveOptions) (string, error) {
	path := ref
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand ~: %w", err)
		}

		path = filepath.Join(home, path[2:])
	}

	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}

		path = filepath.Join(wd, path)
	}

	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("extension not found at %s: %w", path, err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("extension path %s is a directory, expected executable", path)
	}

	if info.Mode()&isExecutableMask == 0 {
		return "", fmt.Errorf("extension at %s is not executable", path)
	}

	return path, nil
}
