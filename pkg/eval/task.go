package eval

// Task represents a single evaluation task for an agent
type Task struct {
	// Prompt to send to the agent
	Prompt string `yaml:"prompt"`

	// Setup script to run before executing the agent (optional)
	Setup string `yaml:"setup,omitempty"`

	// Verifier script to run after agent execution to check success (optional)
	Verifier string `yaml:"verifier,omitempty"`

	// Cleanup script to run after evaluation (optional)
	Cleanup string `yaml:"cleanup,omitempty"`

	// Difficulty level (e.g., "easy", "medium", "hard")
	Difficulty string `yaml:"difficulty"`

	// Expected patterns in agent output
	Expect []Expectation `yaml:"expect,omitempty"`
}

// Expectation represents a pattern to match in agent output
type Expectation struct {
	// Contains specifies a regex pattern that should appear in output
	Contains string `yaml:"contains,omitempty"`
}

// Result represents the outcome of a task evaluation
type Result struct {
	// Task that was evaluated
	Task *Task

	// Whether setup script succeeded
	SetupSuccess bool

	// Whether agent execution succeeded
	AgentSuccess bool

	// Agent output
	AgentOutput string

	// Whether expectations were met
	ExpectationsMet bool

	// Whether verifier script succeeded
	VerifierSuccess bool

	// Whether cleanup script succeeded
	CleanupSuccess bool

	// Overall success status
	Success bool

	// Error if any occurred
	Error error
}
