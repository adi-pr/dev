package project

import (
	"fmt"
	"time"

	"github.com/adi-pr/dev/internal/config"
	"github.com/adi-pr/dev/internal/output"
	projectdomain "github.com/adi-pr/dev/internal/project"
	"github.com/spf13/cobra"
)

var cleanupOlderThan string

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Find inactive projects",

	RunE: func(cmd *cobra.Command, args []string) error {
		age, err := projectdomain.ParseAge(cleanupOlderThan)
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		projects, err := projectdomain.Discover(cfg.ProjectRoots)
		if err != nil {
			return err
		}

		cutoff := time.Now().Add(-age)

		candidates, err := projectdomain.FindCleanupCandidates(
			projects,
			cutoff,
		)
		if err != nil {
			return err
		}

		renderCleanupCandidates(candidates, cleanupOlderThan)

		return nil
	},
}

func init() {
	cleanupCmd.Flags().StringVar(
		&cleanupOlderThan,
		"older-than",
		"6m",
		"inactivity threshold (e.g. 30d, 6m, 1y)",
	)
}

func renderCleanupCandidates(
	candidates []projectdomain.CleanupCandidate,
	threshold string,
) {
	fmt.Printf(
		"%s %s\n",
		output.Primary.Render("Project Cleanup"),
		output.Subtle.Render(
			fmt.Sprintf("· inactive > %s", threshold),
		),
	)

	fmt.Println(
		output.Rule.Render(
			"────────────────────────────────────────────────────────────",
		),
	)

	if len(candidates) == 0 {
		fmt.Println(
			output.Success.Render("No inactive projects found."),
		)
		return
	}

	for _, candidate := range candidates {
		name := output.Text.Copy().
			Bold(true).
			Render(candidate.Project.Name)

		age := formatCleanupAge(candidate.LastActivity)

		state := output.Success.Render("● clean")
		if candidate.Dirty {
			state = output.Warning.Render("● dirty")
		}

		fmt.Printf(
			"%s %s %s\n",
			output.PadRight(name, 22),
			output.PadRight(
				output.Subtle.Render(age),
				16,
			),
			state,
		)
	}
}

func formatCleanupAge(lastActivity time.Time) string {
	duration := time.Since(lastActivity)
	days := int(duration.Hours() / 24)

	switch {
	case days >= 365:
		years := days / 365
		return fmt.Sprintf("%dy ago", years)

	case days >= 30:
		months := days / 30
		return fmt.Sprintf("%dmo ago", months)

	case days >= 7:
		weeks := days / 7
		return fmt.Sprintf("%dw ago", weeks)

	default:
		return fmt.Sprintf("%dd ago", days)
	}
}
