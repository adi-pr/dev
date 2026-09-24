package git

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	gitdomain "github.com/adi-pr/dev/internal/git"
	"github.com/adi-pr/dev/internal/output"
	"github.com/spf13/cobra"
)

var (
	sweepRemote  string
	sweepBase    string
	sweepNoFetch bool
	sweepDryRun  bool
	sweepYes     bool
	sweepJSON    bool
	sweepAll     bool
)

var sweepCmd = &cobra.Command{
	Use:   "sweep",
	Short: "Delete local branches merged into the remote main branch",
	Example: `  dev git sweep
  dev git sweep --dry-run
  dev git sweep --base develop --yes
  dev git sweep --all --dry-run`,
	Args: cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {
		if sweepAll {
			return runSweepAll()
		}

		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		root, err := gitdomain.RepoRoot(cwd)
		if err != nil {
			return err
		}

		plan, err := gitdomain.PrepareSweep(
			root,
			sweepRemote,
			sweepBase,
			!sweepNoFetch,
		)
		if err != nil {
			return err
		}

		if sweepJSON {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(plan)
		}

		renderSweepPlan(plan)

		if len(plan.Branches) == 0 || sweepDryRun {
			return nil
		}

		if !sweepYes {
			confirmed, err := confirmSweep(len(plan.Branches))
			if err != nil {
				return err
			}

			if !confirmed {
				return nil
			}
		}

		fmt.Println()

		if deleteSweepBranches(plan, "") > 0 {
			printRestoreHint("git branch <name> <commit>")
		}

		return nil
	},
}

func init() {
	sweepCmd.Flags().StringVar(
		&sweepRemote,
		"remote",
		"origin",
		"remote to compare against",
	)

	sweepCmd.Flags().StringVar(
		&sweepBase,
		"base",
		"",
		"base branch on the remote (default: remote HEAD)",
	)

	sweepCmd.Flags().BoolVar(
		&sweepNoFetch,
		"no-fetch",
		false,
		"skip fetching the remote first",
	)

	sweepCmd.Flags().BoolVarP(
		&sweepDryRun,
		"dry-run",
		"n",
		false,
		"list merged branches without deleting",
	)

	sweepCmd.Flags().BoolVarP(
		&sweepYes,
		"yes",
		"y",
		false,
		"delete without confirmation",
	)

	sweepCmd.Flags().BoolVar(
		&sweepJSON,
		"json",
		false,
		"print merged branches as JSON without deleting",
	)

	sweepCmd.Flags().BoolVarP(
		&sweepAll,
		"all",
		"a",
		false,
		"sweep every project under the configured project roots",
	)
}

func renderSweepPlan(plan gitdomain.SweepPlan) {
	fmt.Printf(
		"%s %s\n",
		output.Primary.Render("Git Sweep"),
		output.Subtle.Render("· merged into "+plan.RemoteBase()),
	)

	fmt.Println(
		output.Rule.Render(strings.Repeat("─", 60)),
	)

	if len(plan.Branches) == 0 {
		fmt.Println(
			output.Success.Render("No merged branches to sweep."),
		)
		return
	}

	renderSweepBranches(plan.Branches, "")
}

func renderSweepBranches(branches []gitdomain.SweepBranch, indent string) {
	for _, branch := range branches {
		fmt.Printf(
			"%s%s %s %s\n",
			indent,
			output.PadRight(output.Branch.Render(branch.Name), 32),
			output.PadRight(
				output.Subtle.Render(shortCommit(branch.Commit)),
				10,
			),
			renderUpstream(branch),
		)
	}
}

func renderUpstream(branch gitdomain.SweepBranch) string {
	switch {
	case branch.Upstream == "":
		return output.Subtle.Render("local only")

	case branch.Gone:
		return output.Muted.Render("remote deleted")

	default:
		return output.Muted.Render(branch.Upstream)
	}
}

func confirmSweep(count int) (bool, error) {
	noun := "branches"
	if count == 1 {
		noun = "branch"
	}

	fmt.Printf(
		"\nDelete %d %s? %s ",
		count,
		noun,
		output.Muted.Render("[y/N]"),
	)

	input, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, err
	}

	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return true, nil

	default:
		return false, nil
	}
}

// deleteSweepBranches deletes each planned branch, printing one line per
// branch, and returns how many were deleted.
func deleteSweepBranches(plan gitdomain.SweepPlan, indent string) int {
	deleted := 0

	for _, branch := range plan.Branches {
		if err := gitdomain.DeleteMerged(plan, branch); err != nil {
			fmt.Printf(
				"%s%s %s\n",
				indent,
				output.Error.Render("✗"),
				err,
			)
			continue
		}

		deleted++

		fmt.Printf(
			"%s%s Deleted %s %s\n",
			indent,
			output.Success.Render("✓"),
			output.Branch.Render(branch.Name),
			output.Subtle.Render("("+shortCommit(branch.Commit)+")"),
		)
	}

	return deleted
}

func printRestoreHint(command string) {
	fmt.Println()
	fmt.Println(
		output.Subtle.Render("Restore a branch with: " + command),
	)
}

func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}

	return commit
}
