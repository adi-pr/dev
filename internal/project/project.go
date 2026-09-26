package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adi-pr/dev/internal/git"
)

type Project struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Git  bool   `json:"git"`
}

type Status struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Git        bool       `json:"git"`
	Branch     string     `json:"branch,omitempty"`
	Dirty      bool       `json:"dirty"`
	LastCommit *time.Time `json:"last_commit,omitempty"`
	Error      string     `json:"error,omitempty"`
}

func Discover(roots []string) ([]Project, error) {
	var projects []Project

	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, fmt.Errorf("read project root %s: %w", root, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			path := filepath.Join(root, entry.Name())

			_, err := os.Stat(filepath.Join(path, ".git"))
			isGit := err == nil

			projects = append(projects, Project{
				Name: entry.Name(),
				Path: path,
				Git:  isGit,
			})
		}
	}

	sort.Slice(projects, func(i, j int) bool {
		return strings.ToLower(projects[i].Name) <
			strings.ToLower(projects[j].Name)
	})

	return projects, nil
}

func Find(projects []Project, name string) (Project, error) {
	for _, project := range projects {
		if strings.EqualFold(project.Name, name) {
			return project, nil
		}
	}

	return Project{}, fmt.Errorf("project %q not found", name)
}

// Containing returns the project whose directory is path or one of its
// parents. Project paths are resolved through symlinks first, since callers
// may pass real paths such as a process's working directory.
func Containing(projects []Project, path string) (Project, bool) {
	var (
		match   Project
		longest int
	)

	for _, p := range projects {
		root := p.Path

		if resolved, err := filepath.EvalSymlinks(root); err == nil {
			root = resolved
		}

		inside := path == root ||
			strings.HasPrefix(path, root+string(filepath.Separator))

		// Prefer the deepest match in case project roots are nested.
		if inside && len(root) > longest {
			match, longest = p, len(root)
		}
	}

	return match, longest > 0
}

func GetStatus(p Project) Status {
	status := Status{
		Name: p.Name,
		Path: p.Path,
		Git:  p.Git,
	}

	if !p.Git {
		return status
	}

	// A broken repository is reported on its own row rather than failing
	// the whole listing.
	if err := readStatus(p, &status); err != nil {
		status.Error = err.Error()
	}

	return status
}

func readStatus(p Project, status *Status) error {
	branch, err := git.CurrentBranch(p.Path)
	if err != nil {
		return err
	}

	tree, err := git.Status(p.Path)
	if err != nil {
		return err
	}

	lastCommit, ok, err := git.LastCommitTime(p.Path)
	if err != nil {
		return err
	}

	status.Branch = branch
	status.Dirty = tree.Dirty

	if ok {
		status.LastCommit = &lastCommit
	}

	return nil
}
