package cmd

import (
	"fmt"
	"os"

	"github.com/adi-pr/dev/cmd/git"
	"github.com/adi-pr/dev/cmd/project"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dev",
	Short: "Personal development environment CLI",
}

func init() {
	rootCmd.AddCommand(project.Cmd)
	rootCmd.AddCommand(git.Cmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
