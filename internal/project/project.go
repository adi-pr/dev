package project

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Project struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Git  bool   `json:"git"`
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