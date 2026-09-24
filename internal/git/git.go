package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// run executes git in dir and returns trimmed stdout. On failure the error
// includes git's stderr so callers can surface a useful message.
func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}

		return "", fmt.Errorf(
			"git %s: %s",
			strings.Join(args, " "),
			message,
		)
	}

	return strings.TrimSpace(string(out)), nil
}

// RepoRoot returns the top-level directory of the repository containing dir.
func RepoRoot(dir string) (string, error) {
	root, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}

	return root, nil
}
