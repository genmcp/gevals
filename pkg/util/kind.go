package util

import (
	"encoding/json"
	"fmt"
)

// UnmarshalWithKind unmarshals JSON data into target and validates the "kind" field
// matches the expected kind value. The target parameter should be a pointer to the
// struct being unmarshalled.
func UnmarshalWithKind(data []byte, target any, expectedKind string) error {
	// First unmarshal into a temporary struct to check the kind
	tmp := struct {
		Kind string `json:"kind"`
	}{}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	if tmp.Kind != expectedKind {
		return fmt.Errorf("cannot decode kind '%s' as kind '%s'", tmp.Kind, expectedKind)
	}

	// Now unmarshal into the actual target
	return json.Unmarshal(data, target)
}
