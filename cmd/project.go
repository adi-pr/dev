package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/adi-pr/dev/internal/config"
	"github.com/adi-pr/dev/internal/output"
	"github.com/adi-pr/dev/internal/project"
	"github.com/spf13/cobra"
)

var projectJSON bool
var projectStatusJSON bool
var projectInfoJSON bool

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Work with local development projects",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Projects",

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
			displayPath := p.Path

			if home, err := os.UserHomeDir(); err == nil {
				displayPath = strings.Replace(displayPath, home, "~", 1)
			}

			name := output.Text.Copy().Bold(true).Render(p.Name)
			path := output.Subtle.Render(displayPath)

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

var projectOpenCmd = &cobra.Command{
	Use:   "open <name>",
	Short: "Open a project",
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

		if projectStatusJSON {
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

var projectInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show project information",
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

		info := project.GetInfo(p)

		if projectInfoJSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", " ")
			return encoder.Encode(info)
		}

		renderProjectInfo(info)

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

func renderState(dirty bool) string {
	if dirty {
		return output.Warning.Render("● dirty")
	}

	return output.Success.Render("● clean")
}

func renderProjectInfo(info project.Info) {
	fmt.Printf(
		"%s %s\n",
		output.Primary.Render(info.Name),
		output.Subtle.Render("· project info"),
	)

	fmt.Println(
		output.Rule.Render(
			strings.Repeat("─", 60),
		),
	)

	printInfoRow("Path", shortenPath(info.Path))

	if info.Git {
		state := output.Success.Render("● clean")

		if info.Dirty {
			state = output.Warning.Render("● dirty")
		}

		git := output.Branch.Render(info.Branch) + "  " + state
		printInfoRow("Git", git)
	}

	if info.Remote != "" {
		printInfoRow("Remote", info.Remote)
	}

	if info.Language != "" {
		printInfoRow("Language", info.Language)
	}

	if info.Framework != "" {
		printInfoRow("Framework", info.Framework)
	}

	if info.Environment != "" {
		printInfoRow("Environment", info.Environment)
	}

	if info.PackageTool != "" {
		printInfoRow("Package", info.PackageTool)
	}

	if info.Module != "" {
		printInfoRow("Module", info.Module)
	}
}

func printInfoRow(label string, value string) {
	fmt.Printf(
		"%s %s\n",
		output.PadRight(
			output.Subtle.Render(label),
			14,
		),
		output.Text.Render(value),
	)
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return strings.Replace(path, home, "~", 1)
}

func init() {
	rootCmd.AddCommand(projectCmd)

	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectOpenCmd)
	projectCmd.AddCommand(projectStatusCmd)
	projectCmd.AddCommand(projectInfoCmd)

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

	projectInfoCmd.Flags().BoolVar(
		&projectInfoJSON,
		"json",
		false,
		"output as JSON",
	)
}
