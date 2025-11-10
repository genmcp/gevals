package agent

// BuiltinAgent defines the interface for built-in agent implementations
type BuiltinAgent interface {
	// Name returns the agent type identifier
	Name() string

	// Description returns a human-readable description
	Description() string

	// RequiresModel returns true if this agent requires a model parameter
	RequiresModel() bool

	// GetDefaults returns the default AgentSpec for this agent type
	GetDefaults(model string) (*AgentSpec, error)

	// ValidateEnvironment checks if the agent binary is available
	ValidateEnvironment() error
}
