package config

import (
	"fmt"

	"github.com/adi-pr/dev/internal/config"
	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the config file location",
	Args:  cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.Path()
		if err != nil {
			return err
		}

		// Unstyled, so it works in $(dev config path).
		fmt.Println(path)

		return nil
	},
}
