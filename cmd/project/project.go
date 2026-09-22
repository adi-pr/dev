package project

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "project",
	Short: "Work with local development projects",
}

func init() {
	Cmd.AddCommand(
		listCmd,
		openCmd,
		statusCmd,
		infoCmd,
	)
}
