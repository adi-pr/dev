package output

import (
	"os"
	"strings"
)

// ShortenPath replaces the home directory with ~ for display.
func ShortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return strings.Replace(path, home, "~", 1)
}
