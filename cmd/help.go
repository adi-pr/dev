package cmd

import (
	"fmt"
	"strings"

	"github.com/adi-pr/dev/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Println()
		fmt.Println(output.Primary.Render(cmd.CommandPath()))

		if cmd.Short != "" {
			fmt.Println(output.Muted.Render(cmd.Short))
		}

		fmt.Println()
		fmt.Println(output.Rule.Render(strings.Repeat("─", 60)))

		if cmd.HasAvailableSubCommands() {
			fmt.Println()
			fmt.Println(output.Text.Copy().Bold(true).Render("Commands"))

			for _, sub := range cmd.Commands() {
				if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
					continue
				}

				fmt.Printf(
					"  %s %s\n",
					output.PadRight(
						output.Primary.Render(sub.Name()),
						18,
					),
					output.Muted.Render(sub.Short),
				)
			}
		}

		if cmd.HasAvailableLocalFlags() || cmd.HasAvailableInheritedFlags() {
			fmt.Println()
			fmt.Println(output.Text.Copy().Bold(true).Render("Flags"))

			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				name := "--" + flag.Name

				if flag.Shorthand != "" {
					name = "-" + flag.Shorthand + ", " + name
				}

				fmt.Printf(
					"  %s %s\n",
					output.PadRight(
						output.Secondary.Render(name),
						22,
					),
					output.Muted.Render(flag.Usage),
				)
			})
		}

		if cmd.HasExample() {
			fmt.Println()
			fmt.Println(output.Text.Copy().Bold(true).Render("Examples"))
			fmt.Println(output.Muted.Render(cmd.Example))
		}

		fmt.Println()
	})
}
