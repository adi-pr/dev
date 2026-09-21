package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

func GetStatus(p Project) Status {
	status := Status{
		Name: p.Name,
		Path: p.Path,
		Git:  p.Git,
	}

	if !p.Git {
		return status
	}

	status.Branch = gitOutput(p.Path, "branch", "--show-current")

	dirty := gitOutput(p.Path, "status", "--porcelain")
	status.Dirty = dirty != ""

	lastCommit := gitOutput(p.Path, "log", "-1", "--format=%cI")

	if lastCommit != "" {
		t, err := time.Parse(time.RFC3339, lastCommit)
		if err == nil {
			status.LastCommit = &t
		}
	}

	return status
}

func gitOutput(path string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}
