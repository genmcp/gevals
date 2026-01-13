package resolver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSource_Scheme(t *testing.T) {
	s := &FileSource{}
	assert.Equal(t, PackageTypeFile, s.Scheme())
}

func TestFileSource_Resolve(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "file-source-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create an executable file
	executablePath := filepath.Join(tmpDir, "ext-executable")
	err = os.WriteFile(executablePath, []byte("#!/bin/sh\necho test"), 0755)
	require.NoError(t, err)

	// Create a non-executable file
	nonExecutablePath := filepath.Join(tmpDir, "ext-noexec")
	err = os.WriteFile(nonExecutablePath, []byte("not executable"), 0644)
	require.NoError(t, err)

	// Create a directory
	dirPath := filepath.Join(tmpDir, "ext-dir")
	err = os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	tt := map[string]struct {
		ref       string
		expectErr bool
		errMsg    string
	}{
		"absolute path to executable": {
			ref:       executablePath,
			expectErr: false,
		},
		"file not found": {
			ref:       filepath.Join(tmpDir, "nonexistent"),
			expectErr: true,
			errMsg:    "extension not found",
		},
		"path is directory": {
			ref:       dirPath,
			expectErr: true,
			errMsg:    "is a directory",
		},
		"file not executable": {
			ref:       nonExecutablePath,
			expectErr: true,
			errMsg:    "is not executable",
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			s := &FileSource{}
			result, err := s.Resolve(context.Background(), tc.ref, ResolveOptions{})

			if tc.expectErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.ref, result)
		})
	}
}
