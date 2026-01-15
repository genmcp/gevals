package resolver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const isExecutableMask = 0111

// windowsExecutableExts contains file extensions that are considered executable on Windows
var windowsExecutableExts = map[string]bool{
	".exe": true,
	".bat": true,
	".cmd": true,
}

type FileSource struct {
	// BasePath is the directory to use when resolving relative paths.
	// If empty, the current working directory is used.
	BasePath string
}

var _ Source = &FileSource{}

func (s *FileSource) Scheme() string {
	return PackageTypeFile
}

func (s *FileSource) Resolve(ctx context.Context, ref string) (string, error) {
	path := ref
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to expand ~: %w", err)
		}

		path = filepath.Join(home, path[2:])
	}

	if !filepath.IsAbs(path) {
		basePath := s.BasePath
		if basePath == "" {
			var err error
			basePath, err = os.Getwd()
			if err != nil {
				return "", fmt.Errorf("failed to get working directory: %w", err)
			}
		}

		path = filepath.Join(basePath, path)
	}

	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("extension not found at %s: %w", path, err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("extension path %s is a directory, expected executable", path)
	}

	if !isExecutable(path, info) {
		return "", fmt.Errorf("extension at %s is not executable", path)
	}

	return path, nil
}

// isExecutable checks if a file is executable based on the current platform
func isExecutable(path string, info os.FileInfo) bool {
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return windowsExecutableExts[ext]
	}
	return info.Mode()&isExecutableMask != 0
}
