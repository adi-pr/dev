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

var infoJSON bool

var infoCmd = &cobra.Command{
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

		if infoJSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", " ")
			return encoder.Encode(info)
		}

		renderProjectInfo(info)

		return nil
	},
}

func init() {
	infoCmd.Flags().BoolVar(
		&infoJSON,
		"json",
		false,
		"output as JSON",
	)
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

func printInfoRow(label, value string) {
	fmt.Printf(
		"%s %s\n",
		output.PadRight(
			output.Subtle.Render(label),
			14,
		),
		output.Text.Render(value),
	)
}
