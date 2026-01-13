package resolver

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGithubRef(t *testing.T) {
	tt := map[string]struct {
		ref             string
		expectedOwner   string
		expectedRepo    string
		expectedVersion string
		expectErr       bool
	}{
		"simple owner/repo": {
			ref:             "myorg/myext",
			expectedOwner:   "myorg",
			expectedRepo:    "myext",
			expectedVersion: "latest",
			expectErr:       false,
		},
		"with version": {
			ref:             "myorg/myext@v1.0.0",
			expectedOwner:   "myorg",
			expectedRepo:    "myext",
			expectedVersion: "v1.0.0",
			expectErr:       false,
		},
		"with semver": {
			ref:             "genmcp/ext-kubernetes@v2.3.4",
			expectedOwner:   "genmcp",
			expectedRepo:    "ext-kubernetes",
			expectedVersion: "v2.3.4",
			expectErr:       false,
		},
		"empty version after @": {
			ref:       "myorg/myext@",
			expectErr: true,
		},
		"missing repo": {
			ref:       "myorg",
			expectErr: true,
		},
		"too many parts": {
			ref:       "myorg/myext/extra",
			expectErr: true,
		},
		"empty owner": {
			ref:       "/myext",
			expectErr: true,
		},
		"empty repo": {
			ref:       "myorg/",
			expectErr: true,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			owner, repo, version, err := parseGithubRef(tc.ref)

			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedOwner, owner)
			assert.Equal(t, tc.expectedRepo, repo)
			assert.Equal(t, tc.expectedVersion, version)
		})
	}
}

func TestParsePlatform(t *testing.T) {
	tt := map[string]struct {
		platform     string
		expectedGoos string
		expectedArch string
	}{
		"empty uses runtime": {
			platform:     "",
			expectedGoos: runtime.GOOS,
			expectedArch: runtime.GOARCH,
		},
		"darwin-arm64": {
			platform:     "darwin-arm64",
			expectedGoos: "darwin",
			expectedArch: "arm64",
		},
		"linux-amd64": {
			platform:     "linux-amd64",
			expectedGoos: "linux",
			expectedArch: "amd64",
		},
		"invalid format uses runtime": {
			platform:     "invalid",
			expectedGoos: runtime.GOOS,
			expectedArch: runtime.GOARCH,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			goos, goarch := parsePlatform(tc.platform)
			assert.Equal(t, tc.expectedGoos, goos)
			assert.Equal(t, tc.expectedArch, goarch)
		})
	}
}
