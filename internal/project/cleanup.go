package project

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CleanupCandidate struct {
	Project      Project       `json:"project"`
	LastActivity time.Time     `json:"last_activity"`
	Safety       CleanupSafety `json:"safety"`
}

type CleanupSafety struct {
	Dirty         bool `json:"dirty"`
	HasRemote     bool `json:"has_remote"`
	HasUpstream   bool `json:"has_upstream"`
	AheadOfRemote int  `json:"ahead_of_remote"`
	Untracked     int  `json:"untracked"`
}

var ignoredActivityDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	".venv":        true,
	"venv":         true,
	".next":        true,
	"dist":         true,
	"build":        true,
	"target":       true,
	"__pycache__":  true,
}

func FindCleanupCandidates(
	projects []Project,
	olderThan time.Time,
) ([]CleanupCandidate, error) {
	var candidates []CleanupCandidate

	for _, p := range projects {
		candidate, err := NewCleanupCandidate(p)
		if err != nil {
			return nil, err
		}

		if candidate.LastActivity.After(olderThan) {
			continue
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

func NewCleanupCandidate(p Project) (CleanupCandidate, error) {
	lastActivity, err := LastActivity(p)
	if err != nil {
		return CleanupCandidate{}, fmt.Errorf(
			"check activity for %s: %w",
			p.Name,
			err,
		)
	}

	return CleanupCandidate{
		Project:      p,
		LastActivity: lastActivity,
		Safety:       CheckCleanupSafety(p),
	}, nil
}

func LastActivity(p Project) (time.Time, error) {
	var latest time.Time

	err := filepath.WalkDir(
		p.Path,
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if path == p.Path {
				return nil
			}

			if entry.IsDir() && ignoredActivityDirs[entry.Name()] {
				return filepath.SkipDir
			}

			if entry.IsDir() {
				return nil
			}

			info, err := entry.Info()
			if err != nil {
				return err
			}

			if info.ModTime().After(latest) {
				latest = info.ModTime()
			}

			return nil
		},
	)

	if err != nil {
		return time.Time{}, err
	}

	if p.Git {
		value := gitOutput(p.Path, "log", "-1", "--format=%cI")

		if value != "" {
			commitTime, err := time.Parse(time.RFC3339, value)
			if err == nil && commitTime.After(latest) {
				latest = commitTime
			}
		}
	}

	if latest.IsZero() {
		info, err := os.Stat(p.Path)
		if err != nil {
			return time.Time{}, err
		}

		latest = info.ModTime()
	}

	return latest, nil
}

func ParseAge(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))

	if len(value) < 2 {
		return 0, fmt.Errorf("invalid age %q", value)
	}

	number := value[:len(value)-1]
	unit := value[len(value)-1]

	var amount int

	_, err := fmt.Sscanf(number, "%d", &amount)
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf("invalid age %q", value)
	}

	switch unit {
	case 'd':
		return time.Duration(amount) * 24 * time.Hour, nil

	case 'w':
		return time.Duration(amount) * 7 * 24 * time.Hour, nil

	case 'm':
		// Approximation is perfectly adequate for cleanup thresholds.
		return time.Duration(amount) * 30 * 24 * time.Hour, nil

	case 'y':
		return time.Duration(amount) * 365 * 24 * time.Hour, nil

	default:
		return 0, fmt.Errorf(
			"invalid age unit %q; use d, w, m, or y",
			string(unit),
		)
	}
}

func Archive(
	candidate CleanupCandidate,
	archiveRoot string,
) (string, error) {
	if archiveRoot == "" {
		return "", fmt.Errorf("archive root is not configured")
	}

	source := candidate.Project.Path
	destination := filepath.Join(
		archiveRoot,
		filepath.Base(source),
	)

	// Never overwrite an existing archive.
	if _, err := os.Stat(destination); err == nil {
		return "", fmt.Errorf(
			"archive destination already exists: %s",
			destination,
		)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf(
			"check archive destination: %w",
			err,
		)
	}

	if err := os.MkdirAll(archiveRoot, 0755); err != nil {
		return "", fmt.Errorf(
			"create archive directory: %w",
			err,
		)
	}

	if err := os.Rename(source, destination); err != nil {
		return "", fmt.Errorf(
			"archive %s: %w",
			candidate.Project.Name,
			err,
		)
	}

	return destination, nil
}

func CheckCleanupSafety(p Project) CleanupSafety {
	if !p.Git {
		return CleanupSafety{}
	}

	safety := CleanupSafety{}

	status := gitOutput(
		p.Path,
		"status",
		"--porcelain",
	)

	if status != "" {
		safety.Dirty = true

		for _, line := range strings.Split(status, "\n") {
			if strings.HasPrefix(line, "??") {
				safety.Untracked++
			}
		}
	}

	remote := gitOutput(
		p.Path,
		"remote",
	)

	safety.HasRemote = remote != ""

	upstream := gitOutput(
		p.Path,
		"rev-parse",
		"--abbrev-ref",
		"--symbolic-full-name",
		"@{upstream}",
	)

	if upstream == "" {
		return safety
	}

	safety.HasUpstream = true

	ahead := gitOutput(
		p.Path,
		"rev-list",
		"--count",
		"@{upstream}..HEAD",
	)

	if ahead != "" {
		fmt.Sscanf(ahead, "%d", &safety.AheadOfRemote)
	}

	return safety
}

type CleanupRisk int

const (
	CleanupSafe CleanupRisk = iota
	CleanupReview
	CleanupUnsafe
)

func (c CleanupCandidate) Risk() CleanupRisk {
	if !c.Project.Git {
		return CleanupReview
	}

	if c.Safety.Dirty {
		return CleanupUnsafe
	}

	if c.Safety.AheadOfRemote > 0 {
		return CleanupUnsafe
	}

	if !c.Safety.HasRemote {
		return CleanupReview
	}

	if !c.Safety.HasUpstream {
		return CleanupReview
	}

	return CleanupSafe
}