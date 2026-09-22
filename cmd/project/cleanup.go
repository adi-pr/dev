package project

import (
	"bufio"
	"fmt"
	"os"
	"strings"
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
	Args:  cobra.MaximumNArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		projects, err := projectdomain.Discover(cfg.ProjectRoots)
		if err != nil {
			return err
		}

		var candidates []projectdomain.CleanupCandidate

		// Explicit project: age does not matter.
		if len(args) == 1 {
			p, err := projectdomain.Find(projects, args[0])
			if err != nil {
				return err
			}

			candidate, err := projectdomain.NewCleanupCandidate(p)
			if err != nil {
				return err
			}

			candidates = []projectdomain.CleanupCandidate{
				candidate,
			}
		} else {
			// Automatic discovery: use inactivity threshold.
			age, err := projectdomain.ParseAge(cleanupOlderThan)
			if err != nil {
				return err
			}

			cutoff := time.Now().Add(-age)

			candidates, err = projectdomain.FindCleanupCandidates(
				projects,
				cutoff,
			)
			if err != nil {
				return err
			}
		}

		targeted := len(args) == 1

		renderCleanupCandidates(candidates, cleanupOlderThan, targeted)

		if len(candidates) == 0 {
			return nil
		}

		return runCleanup(
			candidates,
			cfg.ArchiveRoot,
		)
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
	targeted bool,
) {
	subtitle := fmt.Sprintf(
		"· inactive > %s",
		threshold,
	)

	if targeted {
		subtitle = "· targeted"
	}

	fmt.Printf(
		"%s %s\n",
		output.Primary.Render("Project Cleanup"),
		output.Subtle.Render(subtitle),
	)

	fmt.Println(
		output.Rule.Render(
			"────────────────────────────────────────────────────────────",
		),
	)

	if len(candidates) == 0 {
		fmt.Println(
			output.Success.Render(
				"No inactive projects found.",
			),
		)
		return
	}

	for _, candidate := range candidates {
		name := output.Text.Copy().
			Bold(true).
			Render(candidate.Project.Name)

		age := formatCleanupAge(candidate.LastActivity)

		state := renderCleanupState(candidate)

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

func runCleanup(
	candidates []projectdomain.CleanupCandidate,
	archiveRoot string,
) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()

	for _, candidate := range candidates {
		action, err := promptCleanupAction(
			reader,
			candidate,
		)
		if err != nil {
			return err
		}

		switch action {
		case "archive":
			destination, err := projectdomain.Archive(
				candidate,
				archiveRoot,
			)
			if err != nil {
				fmt.Printf(
					"%s %s\n",
					output.Error.Render("✗"),
					err,
				)
				continue
			}

			fmt.Printf(
				"%s Archived %s\n",
				output.Success.Render("✓"),
				output.Text.Copy().
					Bold(true).
					Render(candidate.Project.Name),
			)

			fmt.Printf(
				"  %s\n",
				output.Subtle.Render(
					shortenPath(destination),
				),
			)

		case "skip":
			continue

		case "quit":
			return nil
		}
	}

	return nil
}

func promptCleanupAction(
	reader *bufio.Reader,
	candidate projectdomain.CleanupCandidate,
) (string, error) {
	fmt.Println(
		output.Text.Copy().
			Bold(true).
			Render(candidate.Project.Name),
	)

	fmt.Printf(
		"  %s %s\n",
		output.Subtle.Render("Status"),
		renderCleanupState(candidate),
	)

	for {
		switch candidate.Risk() {
		case projectdomain.CleanupSafe:
			fmt.Printf(
				"%s %s %s",
				output.Primary.Render("[a]rchive"),
				output.Error.Render("[d]elete"),
				output.Muted.Render("[s]kip"),
			)

		case projectdomain.CleanupReview,
			projectdomain.CleanupUnsafe:
			fmt.Printf(
				"%s %s",
				output.Primary.Render("[a]rchive"),
				output.Muted.Render("[s]kip"),
			)
		}

		fmt.Printf(
			" %s > ",
			output.Muted.Render("[q]uit"),
		)

		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}

		switch strings.ToLower(strings.TrimSpace(input)) {
		case "a", "archive":
			return "archive", nil

		case "d", "delete":
			if candidate.Risk() != projectdomain.CleanupSafe {
				fmt.Println(
					output.Error.Render(
						"Delete is not available for this project.",
					),
				)
				continue
			}

			return "delete", nil

		case "s", "skip", "":
			return "skip", nil

		case "q", "quit":
			return "quit", nil

		default:
			fmt.Println(
				output.Error.Render(
					"Choose archive, delete, skip, or quit.",
				),
			)
		}
	}
}

func renderCleanupState(
	candidate projectdomain.CleanupCandidate,
) string {
	if !candidate.Project.Git {
		return output.Subtle.Render("no git")
	}

	if candidate.Safety.Dirty {
		if candidate.Safety.Untracked > 0 {
			return output.Error.Render(
				fmt.Sprintf(
					"● dirty · %d untracked",
					candidate.Safety.Untracked,
				),
			)
		}

		return output.Error.Render("● dirty")
	}

	if candidate.Safety.AheadOfRemote > 0 {
		return output.Warning.Render(
			fmt.Sprintf(
				"● %d unpushed",
				candidate.Safety.AheadOfRemote,
			),
		)
	}

	if !candidate.Safety.HasRemote {
		return output.Warning.Render("● no remote")
	}

	if !candidate.Safety.HasUpstream {
		return output.Warning.Render("● no upstream")
	}

	return output.Success.Render("● synced")
}
