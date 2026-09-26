package config

import (
	"fmt"
	"strings"

	"github.com/adi-pr/dev/internal/config"
	"github.com/adi-pr/dev/internal/output"
	"github.com/spf13/cobra"
)

var checkJSON bool

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate the config file",
	Args:  cobra.NoArgs,

	// Failures are already listed; exit non-zero without usage or a
	// repeated "Error:" line.
	SilenceUsage:  true,
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.Path()
		if err != nil {
			return err
		}

		results := config.Check(path)

		if checkJSON {
			if err := output.JSON(results); err != nil {
				return err
			}
		} else {
			renderCheckResults(results)
		}

		return checkFailures(results)
	},
}

func init() {
	checkCmd.Flags().BoolVar(
		&checkJSON,
		"json",
		false,
		"output as JSON",
	)
}

func renderCheckResults(results []config.CheckResult) {
	fmt.Println(output.Primary.Render("Config Check"))

	fmt.Println(
		output.Rule.Render(strings.Repeat("─", 60)),
	)

	for _, result := range results {
		symbol, style := output.Success.Render("✓"), output.Muted

		switch result.Status {
		case config.CheckWarn:
			symbol, style = output.Warning.Render("!"), output.Warning

		case config.CheckFail:
			symbol, style = output.Error.Render("✗"), output.Error
		}

		detail := style.Render(result.Message)

		if result.Path != "" {
			detail = output.Text.Render(output.ShortenPath(result.Path)) +
				output.Subtle.Render(" · ") +
				detail
		}

		fmt.Printf(
			"%s %s %s\n",
			symbol,
			output.PadRight(output.Subtle.Render(result.Name), 14),
			detail,
		)
	}
}

func checkFailures(results []config.CheckResult) error {
	failed := 0

	for _, result := range results {
		if result.Status == config.CheckFail {
			failed++
		}
	}

	switch failed {
	case 0:
		return nil

	case 1:
		return fmt.Errorf("config check: 1 problem")

	default:
		return fmt.Errorf("config check: %d problems", failed)
	}
}
