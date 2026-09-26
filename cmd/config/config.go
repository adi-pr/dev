package config

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect and edit the dev config",
}

func init() {
	Cmd.AddCommand(
		pathCmd,
		editCmd,
		checkCmd,
	)
}
