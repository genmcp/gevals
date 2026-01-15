package client

import (
	"context"
	"errors"
	"testing"

	"github.com/genmcp/gevals/pkg/extension"
	"github.com/genmcp/gevals/pkg/extension/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockResolver implements resolver.Resolver for testing
type mockResolver struct {
	paths  map[string]string
	errors map[string]error
}

func (m *mockResolver) Resolve(ctx context.Context, pkg string) (string, error) {
	if err, ok := m.errors[pkg]; ok {
		return "", err
	}
	if path, ok := m.paths[pkg]; ok {
		return path, nil
	}
	return "", errors.New("package not found")
}

// mockClient implements Client for testing
type mockClient struct {
	manifest    *protocol.InitializeResult
	startErr    error
	executeErr  error
	shutdownErr error
	started     bool
	shutdown    bool
}

func (m *mockClient) Start(ctx context.Context, params *protocol.InitializeParams) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *mockClient) Execute(ctx context.Context, params *protocol.ExecuteParams) (*protocol.ExecuteResult, error) {
	if m.executeErr != nil {
		return nil, m.executeErr
	}
	return &protocol.ExecuteResult{Success: true}, nil
}

func (m *mockClient) Manifest() *protocol.InitializeResult {
	return m.manifest
}

func (m *mockClient) Shutdown(ctx context.Context) error {
	m.shutdown = true
	return m.shutdownErr
}

func TestExtensionManager_Register(t *testing.T) {
	tt := map[string]struct {
		firstAlias  string
		secondAlias string
		expectErr   bool
	}{
		"register new alias": {
			firstAlias: "k8s",
			expectErr:  false,
		},
		"register duplicate alias fails": {
			firstAlias:  "k8s",
			secondAlias: "k8s",
			expectErr:   true,
		},
		"register different aliases succeeds": {
			firstAlias:  "k8s",
			secondAlias: "db",
			expectErr:   false,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			resolver := &mockResolver{paths: make(map[string]string)}
			manager := NewManager(resolver, ExtensionOptions{})

			spec := &extension.ExtensionSpec{Package: "github.com/test/ext"}

			err := manager.Register(tc.firstAlias, spec)
			require.NoError(t, err)

			if tc.secondAlias != "" {
				err = manager.Register(tc.secondAlias, spec)
				if tc.expectErr {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "already registered")
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}

func TestExtensionManager_Has(t *testing.T) {
	tt := map[string]struct {
		registered []string
		checkAlias string
		expected   bool
	}{
		"has registered alias": {
			registered: []string{"k8s"},
			checkAlias: "k8s",
			expected:   true,
		},
		"does not have unregistered alias": {
			registered: []string{"k8s"},
			checkAlias: "db",
			expected:   false,
		},
		"empty manager": {
			registered: []string{},
			checkAlias: "k8s",
			expected:   false,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			resolver := &mockResolver{paths: make(map[string]string)}
			manager := NewManager(resolver, ExtensionOptions{})

			for _, alias := range tc.registered {
				spec := &extension.ExtensionSpec{Package: "github.com/test/" + alias}
				err := manager.Register(alias, spec)
				require.NoError(t, err)
			}

			result := manager.Has(tc.checkAlias)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtensionManager_Get_Errors(t *testing.T) {
	tt := map[string]struct {
		registered    map[string]*extension.ExtensionSpec
		resolverPaths map[string]string
		resolverErrs  map[string]error
		getAlias      string
		expectErr     bool
		errMsg        string
	}{
		"alias not registered": {
			registered: map[string]*extension.ExtensionSpec{},
			getAlias:   "unknown",
			expectErr:  true,
			errMsg:     "no extension registered",
		},
		"resolver fails": {
			registered: map[string]*extension.ExtensionSpec{
				"k8s": {Package: "github.com/test/k8s"},
			},
			resolverErrs: map[string]error{
				"github.com/test/k8s": errors.New("download failed"),
			},
			getAlias:  "k8s",
			expectErr: true,
			errMsg:    "download failed",
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			resolver := &mockResolver{
				paths:  tc.resolverPaths,
				errors: tc.resolverErrs,
			}
			if resolver.paths == nil {
				resolver.paths = make(map[string]string)
			}
			if resolver.errors == nil {
				resolver.errors = make(map[string]error)
			}

			manager := NewManager(resolver, ExtensionOptions{})

			for alias, spec := range tc.registered {
				err := manager.Register(alias, spec)
				require.NoError(t, err)
			}

			_, err := manager.Get(context.Background(), tc.getAlias)

			if tc.expectErr {
				assert.Error(t, err)
				if tc.errMsg != "" {
					assert.Contains(t, err.Error(), tc.errMsg)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestManagerContext(t *testing.T) {
	tt := map[string]struct {
		addToContext bool
		expectFound  bool
	}{
		"manager in context": {
			addToContext: true,
			expectFound:  true,
		},
		"manager not in context": {
			addToContext: false,
			expectFound:  false,
		},
	}

	for tn, tc := range tt {
		t.Run(tn, func(t *testing.T) {
			ctx := context.Background()

			if tc.addToContext {
				resolver := &mockResolver{paths: make(map[string]string)}
				manager := NewManager(resolver, ExtensionOptions{})
				ctx = ManagerToContext(ctx, manager)
			}

			retrieved, ok := ManagerFromContext(ctx)

			assert.Equal(t, tc.expectFound, ok)
			if tc.expectFound {
				assert.NotNil(t, retrieved)
			} else {
				assert.Nil(t, retrieved)
			}
		})
	}
}
