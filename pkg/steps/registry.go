package steps

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Parser func(raw json.RawMessage) (StepRunner, error)

type Registry struct {
	mu      sync.RWMutex
	parsers map[string]Parser
}

func (r *Registry) Register(stepType string, parser Parser) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.parsers[stepType]
	if exists {
		return fmt.Errorf("a parser already exists for type '%s'", stepType)
	}

	r.parsers[stepType] = parser

	return nil
}

func (r *Registry) Parse(cfg StepConfig) (StepRunner, error) {
	if len(cfg) != 1 {
		return nil, fmt.Errorf("each step must have exactly one type")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for stepType, stepCfg := range cfg {
		parser, ok := r.parsers[stepType]
		if !ok {
			return nil, fmt.Errorf("unknown step type '%s'", stepType)
		}

		runner, err := parser(stepCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to parse step: %w", err)
		}

		return runner, nil
	}

	return nil, fmt.Errorf("no step type found")
}
