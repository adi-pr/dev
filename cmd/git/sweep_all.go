package git

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/adi-pr/dev/internal/config"
	gitdomain "github.com/adi-pr/dev/internal/git"
	"github.com/adi-pr/dev/internal/output"
	"github.com/adi-pr/dev/internal/project"
)

// sweepConcurrency bounds how many projects are fetched at once.
const sweepConcurrency = 6

type sweepResult struct {
	Project string               `json:"project"`
	Plan    *gitdomain.SweepPlan `json:"plan,omitempty"`
	Error   string               `json:"error,omitempty"`
}

func runSweepAll() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	projects, err := project.Discover(cfg.ProjectRoots)
	if err != nil {
		return err
	}

	results, skipped := planAllSweeps(projects)

	if sweepJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(results)
	}

	total := renderSweepAll(results, skipped)

	if total == 0 || sweepDryRun {
		return nil
	}

	if !sweepYes {
		confirmed, err := confirmSweep(total)
		if err != nil {
			return err
		}

		if !confirmed {
			return nil
		}
	}

	deleted := 0

	for _, result := range results {
		if result.Plan == nil || len(result.Plan.Branches) == 0 {
			continue
		}

		fmt.Println()
		fmt.Println(output.Text.Copy().Bold(true).Render(result.Project))

		deleted += deleteSweepBranches(*result.Plan, "  ")
	}

	if deleted > 0 {
		printRestoreHint("git -C <project> branch <name> <commit>")
	}

	return nil
}

// planAllSweeps plans a sweep for every git project in parallel. Projects
// without the remote are counted as skipped; other failures are kept as
// per-project errors so one broken repository doesn't stop the rest.
func planAllSweeps(projects []project.Project) ([]sweepResult, int) {
	results := make([]*sweepResult, len(projects))
	sem := make(chan struct{}, sweepConcurrency)

	var wg sync.WaitGroup

	for i, p := range projects {
		if !p.Git {
			continue
		}

		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			plan, err := gitdomain.PrepareSweep(
				p.Path,
				sweepRemote,
				sweepBase,
				!sweepNoFetch,
			)

			switch {
			case errors.Is(err, gitdomain.ErrNoRemote):
				return

			case err != nil:
				results[i] = &sweepResult{Project: p.Name, Error: err.Error()}

			default:
				results[i] = &sweepResult{Project: p.Name, Plan: &plan}
			}
		})
	}

	wg.Wait()

	planned := make([]sweepResult, 0, len(projects))
	skipped := 0

	for i, result := range results {
		if result == nil {
			if projects[i].Git {
				skipped++
			}
			continue
		}

		planned = append(planned, *result)
	}

	return planned, skipped
}

// renderSweepAll prints projects that have merged branches or errors, then a
// summary line. It returns the total number of branches to delete.
func renderSweepAll(results []sweepResult, skipped int) int {
	fmt.Printf(
		"%s %s\n",
		output.Primary.Render("Git Sweep"),
		output.Subtle.Render("· all projects"),
	)

	fmt.Println(
		output.Rule.Render(strings.Repeat("─", 60)),
	)

	total, clean, failed := 0, 0, 0

	for _, result := range results {
		name := output.Text.Copy().Bold(true).Render(result.Project)

		if result.Error != "" {
			failed++

			fmt.Printf("%s %s\n", name, output.Error.Render("✗"))

			// Render line by line; lipgloss pads multi-line strings into
			// a block.
			for _, line := range strings.Split(result.Error, "\n") {
				if line = strings.TrimSpace(line); line != "" {
					fmt.Printf("  %s\n", output.Subtle.Render(line))
				}
			}
			continue
		}

		if len(result.Plan.Branches) == 0 {
			clean++
			continue
		}

		total += len(result.Plan.Branches)

		fmt.Printf(
			"%s %s\n",
			name,
			output.Subtle.Render("· merged into "+result.Plan.RemoteBase()),
		)

		renderSweepBranches(result.Plan.Branches, "  ")
	}

	if total == 0 && failed == 0 {
		fmt.Println(
			output.Success.Render("No merged branches to sweep."),
		)
	}

	summary := fmt.Sprintf("%d clean", clean)

	if failed > 0 {
		summary += fmt.Sprintf(" · %d failed", failed)
	}

	if skipped > 0 {
		summary += fmt.Sprintf(" · %d skipped (no %s remote)", skipped, sweepRemote)
	}

	fmt.Println()
	fmt.Println(output.Subtle.Render(summary))

	return total
}
