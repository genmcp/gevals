package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseRef(t *testing.T) {
	tt := map[string]struct {
		input          string
		expectedScheme string
		expectedPath   string
	}{
		"file:// prefix": {
			input:          "file:///usr/local/bin/ext",
			expectedScheme: PackageTypeFile,
			expectedPath:   "/usr/local/bin/ext",
		},
		"absolute path": {
			input:          "/usr/local/bin/ext",
			expectedScheme: PackageTypeFile,
			expectedPath:   "/usr/local/bin/ext",
		},
		"relative path with ./": {
			input:          "./local/ext",
			expectedScheme: PackageTypeFile,
			expectedPath:   "./local/ext",
		},
		"relative path with ../": {
			input:          "../other/ext",
			expectedScheme: PackageTypeFile,
			expectedPath:   "../other/ext",
		},
		"home directory path": {
			input:          "~/bin/ext",
			expectedScheme: PackageTypeFile,
			expectedPath:   "~/bin/ext",
		},
		"github.com reference": {
			input:          "github.com/myorg/myext",
			expectedScheme: PackageTypeGithub,
			expectedPath:   "myorg/myext",
		},
		"github.com reference with version": {
			input:          "github.com/myorg/myext@v1.0.0",
			expectedScheme: PackageTypeGithub,
			expectedPath:   "myorg/myext@v1.0.0",
		},
		"unknown scheme": {
			input:          "someother/reference",
			expectedScheme: PackageTypeUnknown,
			expectedPath:   "someother/reference",
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			scheme, path := parseRef(tc.input)
			assert.Equal(t, tc.expectedScheme, scheme)
			assert.Equal(t, tc.expectedPath, path)
		})
	}
}
