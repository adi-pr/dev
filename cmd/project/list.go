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

var projectJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",

	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		projects, err := project.Discover(cfg.ProjectRoots)
		if err != nil {
			return err
		}

		if projectJSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(projects)
		}

		fmt.Printf(
			"%s %s\n",
			output.Primary.Render("Projects"),
			output.Subtle.Render(
				fmt.Sprintf("· %d", len(projects)),
			),
		)

		fmt.Println(
			output.Rule.Render(
				strings.Repeat("─", 60),
			),
		)

		for _, p := range projects {
			name := output.Text.Copy().
				Bold(true).
				Render(p.Name)

			path := output.Subtle.Render(shortenPath(p.Path))

			git := ""
			if p.Git {
				git = output.Primary.Render("git")
			}

			fmt.Printf(
				"%s %s %s\n",
				output.PadRight(name, 18),
				output.PadRight(path, 40),
				git,
			)
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(
		&projectJSON,
		"json",
		false,
		"output as JSON",
	)
}
