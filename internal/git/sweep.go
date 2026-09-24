package git

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// protectedBranches are never swept, even when fully merged.
var protectedBranches = map[string]bool{
	"main":    true,
	"master":  true,
	"develop": true,
}

type SweepBranch struct {
	Name     string `json:"name"`
	Commit   string `json:"commit"`
	Upstream string `json:"upstream,omitempty"`
	Gone     bool   `json:"upstream_gone"`
}

type SweepPlan struct {
	Root     string        `json:"root"`
	Remote   string        `json:"remote"`
	Base     string        `json:"base"`
	Current  string        `json:"current"`
	Branches []SweepBranch `json:"branches"`
}

// RemoteBase returns the fully qualified remote base ref, e.g. origin/main.
func (p SweepPlan) RemoteBase() string {
	return p.Remote + "/" + p.Base
}

// ErrNoRemote is returned by PrepareSweep when the repository does not have
// the requested remote.
var ErrNoRemote = errors.New("remote not configured")

// PrepareSweep fetches (unless fetch is false), resolves the base branch when
// base is empty, and plans a sweep of the repository at root.
func PrepareSweep(root, remote, base string, fetch bool) (SweepPlan, error) {
	remotes, err := Remotes(root)
	if err != nil {
		return SweepPlan{}, err
	}

	if !slices.Contains(remotes, remote) {
		return SweepPlan{}, fmt.Errorf("%s: %w", remote, ErrNoRemote)
	}

	if fetch {
		if err := Fetch(root, remote); err != nil {
			return SweepPlan{}, err
		}
	}

	if base == "" {
		base, err = DefaultBase(root, remote)
		if err != nil {
			return SweepPlan{}, err
		}
	}

	return PlanSweep(root, remote, base)
}

// Fetch updates remote-tracking refs and prunes deleted remote branches so
// merge detection runs against the remote's current state.
func Fetch(root, remote string) error {
	_, err := run(root, "fetch", "--prune", "--quiet", remote)
	return err
}

// DefaultBase resolves the remote's default branch, falling back to main or
// master when the remote HEAD is not set locally.
func DefaultBase(root, remote string) (string, error) {
	head, err := run(
		root,
		"symbolic-ref",
		"--short",
		"refs/remotes/"+remote+"/HEAD",
	)
	if err == nil && head != "" {
		return strings.TrimPrefix(head, remote+"/"), nil
	}

	for _, candidate := range []string{"main", "master"} {
		exists, err := refExists(root, "refs/remotes/"+remote+"/"+candidate)
		if err != nil {
			return "", err
		}

		if exists {
			return candidate, nil
		}
	}

	return "", fmt.Errorf(
		"could not determine default branch for %s; use --base",
		remote,
	)
}

// PlanSweep lists local branches whose tips are fully contained in
// remote/base. The current branch, the base branch, protected branches and
// branches checked out in other worktrees are excluded.
func PlanSweep(root, remote, base string) (SweepPlan, error) {
	plan := SweepPlan{
		Root:   root,
		Remote: remote,
		Base:   base,
	}

	exists, err := refExists(root, "refs/remotes/"+plan.RemoteBase())
	if err != nil {
		return plan, err
	}

	if !exists {
		return plan, fmt.Errorf(
			"remote branch %s does not exist",
			plan.RemoteBase(),
		)
	}

	plan.Current, _ = run(root, "branch", "--show-current")

	out, err := run(
		root,
		"for-each-ref",
		"--merged="+plan.RemoteBase(),
		"--format=%(refname:short)\t%(objectname)\t%(upstream:short)\t%(upstream:track)\t%(worktreepath)",
		"refs/heads/",
	)
	if err != nil {
		return plan, err
	}

	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		// Output is whitespace-trimmed, so the last line loses its empty
		// trailing fields; pad them back.
		fields := strings.Split(line, "\t")
		for len(fields) < 5 {
			fields = append(fields, "")
		}

		name := fields[0]

		if name == base || name == plan.Current || protectedBranches[name] {
			continue
		}

		// Checked out in another worktree; git refuses to delete it anyway.
		if fields[4] != "" {
			continue
		}

		plan.Branches = append(plan.Branches, SweepBranch{
			Name:     name,
			Commit:   fields[1],
			Upstream: fields[2],
			Gone:     fields[3] == "[gone]",
		})
	}

	return plan, nil
}

// DeleteMerged deletes a branch after re-verifying that the commit it points
// to is still an ancestor of remote/base. The delete is pinned to the planned
// commit, so a branch that moved since planning is left untouched.
func DeleteMerged(plan SweepPlan, branch SweepBranch) error {
	current, err := run(plan.Root, "rev-parse", "refs/heads/"+branch.Name)
	if err != nil {
		return err
	}

	if current != branch.Commit {
		return fmt.Errorf("%s moved since it was checked", branch.Name)
	}

	if _, err := run(
		plan.Root,
		"merge-base",
		"--is-ancestor",
		branch.Commit,
		plan.RemoteBase(),
	); err != nil {
		return fmt.Errorf(
			"%s is not merged into %s",
			branch.Name,
			plan.RemoteBase(),
		)
	}

	// update-ref with an expected old value is atomic: it fails if the
	// branch changed between the check above and the delete.
	if _, err := run(
		plan.Root,
		"update-ref",
		"-d",
		"refs/heads/"+branch.Name,
		branch.Commit,
	); err != nil {
		return err
	}

	// Drop the branch's config section (upstream tracking etc.). Missing
	// sections are not an error worth reporting.
	run(plan.Root, "config", "--remove-section", "branch."+branch.Name)

	return nil
}

func refExists(root, ref string) (bool, error) {
	_, ok, err := probe(root, "show-ref", "--verify", "--quiet", ref)
	return ok, err
}
