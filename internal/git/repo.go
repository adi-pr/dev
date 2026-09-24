package git

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

type WorkingTree struct {
	Dirty     bool
	Untracked int
}

// CurrentBranch returns the checked-out branch, or "" when HEAD is detached.
func CurrentBranch(dir string) (string, error) {
	return run(dir, "branch", "--show-current")
}

// Status summarises uncommitted and untracked changes.
func Status(dir string) (WorkingTree, error) {
	out, err := run(dir, "status", "--porcelain")
	if err != nil {
		return WorkingTree{}, err
	}

	tree := WorkingTree{Dirty: out != ""}

	for _, line := range lines(out) {
		if strings.HasPrefix(line, "??") {
			tree.Untracked++
		}
	}

	return tree, nil
}

func Remotes(dir string) ([]string, error) {
	out, err := run(dir, "remote")
	if err != nil {
		return nil, err
	}

	return lines(out), nil
}

// RemoteURL returns the URL of the named remote, or "" when it does not exist.
func RemoteURL(dir, name string) (string, error) {
	remotes, err := Remotes(dir)
	if err != nil {
		return "", err
	}

	if !slices.Contains(remotes, name) {
		return "", nil
	}

	return run(dir, "remote", "get-url", name)
}

// Upstream returns the current branch's upstream (e.g. origin/main), or ""
// when HEAD is detached, no upstream is configured, or the upstream branch
// no longer exists on the remote.
func Upstream(dir string) (string, error) {
	head, ok, err := probe(dir, "symbolic-ref", "--quiet", "HEAD")
	if err != nil || !ok {
		return "", err
	}

	out, err := run(
		dir,
		"for-each-ref",
		"--format=%(upstream:short)\t%(upstream:track)",
		head,
	)
	if err != nil {
		return "", err
	}

	upstream, track, _ := strings.Cut(out, "\t")
	if track == "[gone]" {
		return "", nil
	}

	return upstream, nil
}

// CommitsAhead counts commits reachable from head but not from base.
func CommitsAhead(dir, base, head string) (int, error) {
	return count(dir, "rev-list", "--count", base+".."+head)
}

// UnpushedCommits counts commits on any local branch that no remote-tracking
// branch contains.
func UnpushedCommits(dir string) (int, error) {
	return count(
		dir,
		"rev-list",
		"--count",
		"--branches",
		"--not",
		"--remotes",
	)
}

func StashCount(dir string) (int, error) {
	out, err := run(dir, "stash", "list")
	if err != nil {
		return 0, err
	}

	return len(lines(out)), nil
}

// LastCommitTime returns the committer date of HEAD. ok is false when the
// repository has no commits yet.
func LastCommitTime(dir string) (t time.Time, ok bool, err error) {
	_, ok, err = probe(dir, "rev-parse", "--verify", "--quiet", "HEAD")
	if err != nil || !ok {
		return time.Time{}, false, err
	}

	out, err := run(dir, "log", "-1", "--format=%cI")
	if err != nil {
		return time.Time{}, false, err
	}

	t, err = time.Parse(time.RFC3339, out)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("parse commit time %q: %w", out, err)
	}

	return t, true, nil
}

func count(dir string, args ...string) (int, error) {
	out, err := run(dir, args...)
	if err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(out)
	if err != nil {
		return 0, fmt.Errorf("parse count %q: %w", out, err)
	}

	return n, nil
}

func lines(out string) []string {
	if out == "" {
		return nil
	}

	return strings.Split(out, "\n")
}
