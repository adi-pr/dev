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
		"%s ",
		output.Primary.Render("[a]rchive"),
	)

	fmt.Printf(
		"%s ",
		output.Muted.Render("[s]kip"),
	)

	fmt.Printf(
		"%s",
		output.Muted.Render("[q]uit"),
	)

	fmt.Print(" > ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	switch strings.ToLower(strings.TrimSpace(input)) {
	case "a", "archive":
		return "archive", nil

	case "s", "skip", "":
		return "skip", nil

	case "q", "quit":
		return "quit", nil

	default:
		fmt.Println(
			output.Error.Render(
				"Choose archive, skip, or quit.",
			),
		)

		return promptCleanupAction(reader, candidate)
	}
}