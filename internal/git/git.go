package git

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Error is a failed git invocation. It carries git's stderr so callers can
// surface a useful message.
type Error struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *Error) Error() string {
	message := e.Stderr
	if message == "" {
		message = e.Err.Error()
	}

	return fmt.Sprintf(
		"git %s: %s",
		strings.Join(e.Args, " "),
		message,
	)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// run executes git in dir and returns trimmed stdout.
func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		return "", &Error{
			Args:   args,
			Stderr: strings.TrimSpace(stderr.String()),
			Err:    err,
		}
	}

	return strings.TrimSpace(string(out)), nil
}

// probe runs a git command whose exit status 1 means "not found" rather than
// failure, as with show-ref --verify, rev-parse --verify and symbolic-ref
// --quiet. Any other failure is returned as an error.
func probe(dir string, args ...string) (string, bool, error) {
	out, err := run(dir, args...)
	if err == nil {
		return out, true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return "", false, nil
	}

	return "", false, err
}

// RepoRoot returns the top-level directory of the repository containing dir.
func RepoRoot(dir string) (string, error) {
	root, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}

	return root, nil
}
