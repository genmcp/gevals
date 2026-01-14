package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"sync"
)

type Parser func(raw json.RawMessage) (StepRunner, error)

type PrefixParser func(suffix string, raw json.RawMessage) (StepRunner, error)

type Registry struct {
	mu            sync.RWMutex
	parsers       map[string]Parser
	prefixParsers map[string]PrefixParser
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

func (r *Registry) RegisterPrefix(prefix string, parser PrefixParser) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.prefixParsers[prefix]
	if exists {
		return fmt.Errorf("a prefix parser already exists for prefix '%s'", prefix)
	}

	r.prefixParsers[prefix] = parser

	return nil
}

func (r *Registry) WithExtensions(ctx context.Context, aliases []string) *Registry {
	r.mu.RLock()
	reg := &Registry{
		parsers:       make(map[string]Parser, len(r.parsers)),
		prefixParsers: make(map[string]PrefixParser, len(r.prefixParsers)),
	}
	maps.Copy(reg.parsers, r.parsers)
	maps.Copy(reg.prefixParsers, r.prefixParsers)
	r.mu.RUnlock()

	for _, alias := range aliases {
		reg.prefixParsers[alias] = NewExtensionParser(ctx, alias)
	}

	return reg
}

func (r *Registry) Parse(cfg StepConfig) (StepRunner, error) {
	if len(cfg) != 1 {
		return nil, fmt.Errorf("each step must have exactly one type")
	}

	for stepType, stepCfg := range cfg {
		if strings.Contains(stepType, ".") {
			return r.parsePrefix(stepType, stepCfg)
		}

		return r.parse(stepType, stepCfg)
	}

	return nil, fmt.Errorf("no step type found")
}

func (r *Registry) parse(stepType string, stepCfg json.RawMessage) (StepRunner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

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

func (r *Registry) parsePrefix(stepType string, stepCfg json.RawMessage) (StepRunner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	parts := strings.SplitN(stepType, ".", 2)

	parser, ok := r.prefixParsers[parts[0]]
	if !ok {
		return nil, fmt.Errorf("unknown step type '%s'", stepType)
	}

	runner, err := parser(parts[1], stepCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse step: %w", err)
	}

	return runner, nil
}
