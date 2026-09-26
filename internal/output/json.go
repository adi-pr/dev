package output

import (
	"encoding/json"
	"os"
)

// JSON writes v to stdout as indented JSON. Every --json flag goes through
// here so all commands share one output format.
func JSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
