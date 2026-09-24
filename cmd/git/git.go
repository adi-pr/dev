package git

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "git",
	Short: "Git workflow helpers",
}

func init() {
	Cmd.AddCommand(
		sweepCmd,
	)
}
