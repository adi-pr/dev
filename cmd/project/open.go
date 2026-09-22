package project

import (
	"os"
	"os/exec"

	"github.com/adi-pr/dev/internal/config"
	"github.com/adi-pr/dev/internal/project"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
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
