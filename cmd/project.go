package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/adi-pr/dev/internal/config"
	"github.com/adi-pr/dev/internal/project"
	"github.com/spf13/cobra"
)

var projectJSON bool

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

func init() {
	rootCmd.AddCommand(projectCmd)

	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectOpenCmd)

	projectListCmd.Flags().BoolVar(
		&projectJSON,
		"json",
		false,
		"output as JSON",
	)
}