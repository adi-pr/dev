package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

type CheckStatus string

const (
	CheckOK   CheckStatus = "ok"
	CheckWarn CheckStatus = "warn"
	CheckFail CheckStatus = "fail"
)

// CheckResult is the outcome of one config check. Path is the file or
// directory the check is about, when there is one.
type CheckResult struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Path    string      `json:"path,omitempty"`
	Message string      `json:"message"`
}

func (r CheckResult) ok(message string) CheckResult {
	r.Status, r.Message = CheckOK, message
	return r
}

func (r CheckResult) warn(message string) CheckResult {
	r.Status, r.Message = CheckWarn, message
	return r
}

func (r CheckResult) fail(message string) CheckResult {
	r.Status, r.Message = CheckFail, message
	return r
}

// Check validates the config file at path. Every check runs even when an
// earlier one fails, unless the file itself cannot be read or parsed.
func Check(path string) []CheckResult {
	file := CheckResult{Name: "config", Path: path}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return []CheckResult{file.fail("does not exist; create it with dev config edit")}
	}

	if err != nil {
		return []CheckResult{file.fail(reason(err))}
	}

	var cfg Config

	if err := json.Unmarshal(data, &cfg); err != nil {
		return []CheckResult{file.fail("invalid JSON: " + err.Error())}
	}

	results := []CheckResult{checkFields(file, data)}

	if len(cfg.ProjectRoots) == 0 {
		results = append(results, CheckResult{Name: "project root"}.fail(
			"no project_roots configured",
		))
	}

	for _, root := range cfg.ProjectRoots {
		results = append(results, checkProjectRoot(root))
	}

	results = append(
		results,
		checkArchiveRoot(cfg.ArchiveRoot, cfg.ProjectRoots),
		checkEditor(cfg.Editor),
	)

	return results
}

// checkFields reports keys Config doesn't know. json.Unmarshal ignores them,
// so a typo such as "archiveRoot" would otherwise silently unset the value.
// Only the first unknown key is reported.
func checkFields(result CheckResult, data []byte) CheckResult {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var cfg Config

	if err := decoder.Decode(&cfg); err != nil {
		return result.warn(
			strings.TrimPrefix(err.Error(), "json: ") + " is ignored",
		)
	}

	return result.ok("valid")
}

func checkProjectRoot(root string) CheckResult {
	result := CheckResult{Name: "project root", Path: root}

	if err := absolute(root); err != nil {
		return result.fail(err.Error())
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return result.fail(reason(err))
	}

	count := 0

	for _, entry := range entries {
		if entry.IsDir() {
			count++
		}
	}

	return result.ok(fmt.Sprintf("%d projects", count))
}

// checkArchiveRoot confirms cleanup can move projects into the archive root.
// Archive creates the directory on demand, so a missing one is fine as long
// as its nearest existing parent is writable.
func checkArchiveRoot(archiveRoot string, projectRoots []string) CheckResult {
	result := CheckResult{Name: "archive root", Path: archiveRoot}

	if archiveRoot == "" {
		return result.warn("not set; project cleanup cannot archive")
	}

	if err := absolute(archiveRoot); err != nil {
		return result.fail(err.Error())
	}

	dir, info, err := existingAncestor(archiveRoot)
	if err != nil {
		return result.fail(reason(err))
	}

	if !info.IsDir() {
		return result.fail(dir + " is not a directory")
	}

	if err := unix.Access(dir, unix.W_OK); err != nil {
		return result.fail(dir + " is not writable")
	}

	// Archive moves projects with os.Rename, which cannot cross filesystems.
	for _, root := range projectRoots {
		rootInfo, err := os.Stat(root)
		if err != nil {
			continue
		}

		if device(rootInfo) != device(info) {
			return result.warn(
				"on a different filesystem from " + root +
					"; archiving will fail",
			)
		}
	}

	if dir != archiveRoot {
		return result.ok("will be created on first archive")
	}

	return result.ok("writable")
}

func checkEditor(editor string) CheckResult {
	result := CheckResult{Name: "editor"}

	name := editor
	if name == "" {
		name = DefaultEditor + " (default)"
		editor = DefaultEditor
	}

	path, err := exec.LookPath(editor)
	if err != nil {
		return result.fail(name + " not found on PATH")
	}

	result.Path = path

	return result.ok(name)
}

// absolute rejects relative paths, which would resolve against whatever
// directory dev happens to run in.
func absolute(path string) error {
	if strings.HasPrefix(path, "~") {
		return errors.New("~ is not expanded; use an absolute path")
	}

	if !filepath.IsAbs(path) {
		return errors.New("must be an absolute path")
	}

	return nil
}

// existingAncestor returns path, or its closest parent that exists.
func existingAncestor(path string) (string, fs.FileInfo, error) {
	for {
		info, err := os.Stat(path)
		if err == nil {
			return path, info, nil
		}

		parent := filepath.Dir(path)

		// ENOTDIR means a parent is a file; keep walking so the caller
		// can name it.
		missing := errors.Is(err, fs.ErrNotExist) ||
			errors.Is(err, syscall.ENOTDIR)

		if !missing || parent == path {
			return "", nil, err
		}

		path = parent
	}
}

func device(info fs.FileInfo) uint64 {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0
	}

	return stat.Dev
}

// reason drops the path from filesystem errors; results already carry it.
func reason(err error) string {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err.Error()
	}

	return err.Error()
}
