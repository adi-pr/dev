package project

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/adi-pr/dev/internal/config"
	"github.com/adi-pr/dev/internal/output"
	"github.com/adi-pr/dev/internal/project"
	"github.com/spf13/cobra"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		projects, err := project.Discover(cfg.ProjectRoots)
		if err != nil {
			return err
		}

		statuses := make([]project.Status, 0, len(projects))

		for _, p := range projects {
			statuses = append(statuses, project.GetStatus(p))
		}

		if statusJSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(statuses)
		}

		fmt.Printf(
			"%s %s\n",
			output.Primary.Render("Project Status"),
			output.Subtle.Render(
				fmt.Sprintf("· %d", len(statuses)),
			),
		)

		fmt.Println(
			output.Rule.Render(
				strings.Repeat("─", 60),
			),
		)

		for _, status := range statuses {
			name := output.Text.Copy().
				Bold(true).
				Render(status.Name)

			if !status.Git {
				fmt.Printf(
					"%s %s %s\n",
					output.PadRight(name, 18),
					output.PadRight(output.Subtle.Render("—"), 15),
					output.Subtle.Render("no git"),
				)

				continue
			}

			if status.Error != "" {
				fmt.Printf(
					"%s %s %s\n",
					output.PadRight(name, 18),
					output.PadRight(output.Subtle.Render("—"), 15),
					output.Error.Render("● git error"),
				)

				continue
			}

			branch := status.Branch
			if branch == "" {
				branch = "detached"
			}

			lastCommit := "—"
			if status.LastCommit != nil {
				lastCommit = formatRelativeTime(*status.LastCommit)
			}

			fmt.Printf(
				"%s %s %s %s\n",
				output.PadRight(name, 18),
				output.PadRight(
					output.Branch.Render(branch),
					15,
				),
				output.PadRight(
					renderState(status.Dirty),
					12,
				),
				output.Subtle.Render(lastCommit),
			)
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(
		&statusJSON,
		"json",
		false,
		"output as JSON",
	)
}
