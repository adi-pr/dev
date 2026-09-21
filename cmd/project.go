package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/adi-pr/dev/internal/config"
	"github.com/adi-pr/dev/internal/project"
	"github.com/spf13/cobra"
)

var projectJSON bool
var projectStatusJSON bool

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Work with local development projects",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known projects",

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

		for _, p := range projects {
			git := ""

			if p.Git {
				git = " [git]"
			}

			fmt.Printf("%-24s %s%s\n", p.Name, p.Path, git)
		}

		return nil
	},
}

var projectOpenCmd = &cobra.Command{
	Use:   "open <name>",
	Short: "Open a project in the configured editor",
	Args:  cobra.ExactArgs(1),

	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		projects, err := project.Discover(cfg.ProjectRoots)
		if err != nil {
			return err
		}

		p, err := project.Find(projects, args[0])
		if err != nil {
			return err
		}

		editor := cfg.Editor
		if editor == "" {
			editor = "code"
		}

		command := exec.Command(editor, p.Path)

		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Stdin = os.Stdin

		return command.Start()
	},
}

var projectStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of known projects",

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

		if projectStatusJSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")

			return encoder.Encode(statuses)
		}

		fmt.Printf(
			"%-24s %-20s %-10s %s\n",
			"PROJECT",
			"BRANCH",
			"STATE",
			"LAST COMMIT",
		)

		for _, status := range statuses {
			if !status.Git {
				fmt.Printf(
					"%-24s %-20s %-10s %s\n",
					status.Name,
					"-",
					"-",
					"-",
				)

				continue
			}

			state := "clean"

			if status.Dirty {
				state = "dirty"
			}

			lastCommit := "-"

			if status.LastCommit != nil {
				lastCommit = formatRelativeTime(*status.LastCommit)
			}

			fmt.Printf(
				"%-24s %-20s %-10s %s\n",
				status.Name,
				status.Branch,
				state,
				lastCommit,
			)
		}

		return nil
	},
}

func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"

	case duration < time.Hour:
		minutes := int(duration.Minutes())

		if minutes == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", minutes)

	case duration < 24*time.Hour:
		hours := int(duration.Hours())

		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)

	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)

		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)

	default:
		return t.Format("2006-01-02")
	}
}

func init() {
	rootCmd.AddCommand(projectCmd)

	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectOpenCmd)
	projectCmd.AddCommand(projectStatusCmd)

	projectListCmd.Flags().BoolVar(
		&projectJSON,
		"json",
		false,
		"output as JSON",
	)

	projectStatusCmd.Flags().BoolVar(
		&projectStatusJSON,
		"json",
		false,
		"output as json",
	)
}
