package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// outputJSONTo writes a value as two-space-indented JSON.
func outputJSONTo(w io.Writer, v any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

// outputAnyJSON outputs any value as JSON.
func outputAnyJSON(v any) error {
	return outputJSONTo(os.Stdout, v)
}

// outputAnyYAML outputs any value as YAML.
func outputAnyYAML(v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Print(string(data))
	return nil
}
