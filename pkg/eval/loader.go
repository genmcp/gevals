package eval

import (
	"os"

	"sigs.k8s.io/yaml"
)

// LoadTask loads a task definition from a YAML file
func LoadTask(filepath string) (*Task, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var task Task
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, err
	}

	return &task, nil
}
