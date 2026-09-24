package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/adi-pr/dev/internal/git"
)

type Info struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Git         bool   `json:"git"`
	Branch      string `json:"branch,omitempty"`
	Dirty       bool   `json:"dirty"`
	Remote      string `json:"remote,omitempty"`
	Language    string `json:"language,omitempty"`
	Framework   string `json:"framework,omitempty"`
	Environment string `json:"environment,omitempty"`
	PackageTool string `json:"package_tool,omitempty"`
	Module      string `json:"module,omitempty"`
}

func GetInfo(p Project) (Info, error) {
	info := Info{
		Name: p.Name,
		Path: p.Path,
		Git:  p.Git,
	}

	if err := detectGit(p, &info); err != nil {
		return Info{}, err
	}

	detectGo(p.Path, &info)
	detectPython(p.Path, &info)
	detectJavaScript(p.Path, &info)

	return info, nil
}

func detectGit(p Project, info *Info) error {
	if !p.Git {
		return nil
	}

	branch, err := git.CurrentBranch(p.Path)
	if err != nil {
		return err
	}

	tree, err := git.Status(p.Path)
	if err != nil {
		return err
	}

	remote, err := git.RemoteURL(p.Path, "origin")
	if err != nil {
		return err
	}

	info.Branch = branch
	info.Dirty = tree.Dirty
	info.Remote = remote

	return nil
}

func detectGo(path string, info *Info) {
	goMod := filepath.Join(path, "go.mod")

	data, err := os.ReadFile(goMod)
	if err != nil {
		return
	}

	info.Language = "Go"

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "module ") {
			info.Module = strings.TrimSpace(
				strings.TrimPrefix(line, "module "),
			)
			break
		}
	}
}

func detectPython(path string, info *Info) {
	if fileExists(filepath.Join(path, "pyproject.toml")) ||
		fileExists(filepath.Join(path, "requirements.txt")) {

		info.Language = "Python"
	}

	if dirExists(filepath.Join(path, ".venv")) {
		info.Environment = ".venv"
	}
}

func detectJavaScript(path string, info *Info) {
	packagePath := filepath.Join(path, "package.json")

	data, err := os.ReadFile(packagePath)
	if err != nil {
		return
	}

	if info.Language == "" {
		info.Language = "JavaScript"
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if json.Unmarshal(data, &pkg) == nil {
		if hasDependency(pkg.Dependencies, "next") ||
			hasDependency(pkg.DevDependencies, "next") {
			info.Framework = "Next.js"
		}
	}

	switch {
	case fileExists(filepath.Join(path, "bun.lock")) ||
		fileExists(filepath.Join(path, "bun.lockb")):
		info.PackageTool = "Bun"

	case fileExists(filepath.Join(path, "pnpm-lock.yaml")):
		info.PackageTool = "pnpm"

	case fileExists(filepath.Join(path, "yarn.lock")):
		info.PackageTool = "Yarn"

	case fileExists(filepath.Join(path, "package-lock.json")):
		info.PackageTool = "npm"
	}
}

func hasDependency(dependencies map[string]string, name string) bool {
	_, exists := dependencies[name]
	return exists
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
