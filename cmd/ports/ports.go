package ports

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/adi-pr/dev/internal/config"
	"github.com/adi-pr/dev/internal/output"
	"github.com/adi-pr/dev/internal/ports"
	"github.com/adi-pr/dev/internal/project"
	"github.com/spf13/cobra"
)

// killTimeout is how long a killed process gets to release its port.
const killTimeout = 5 * time.Second

var (
	portsAll   bool
	portsJSON  bool
	portsYes   bool
	portsForce bool
)

var Cmd = &cobra.Command{
	Use:   "ports [port]",
	Short: "Show listening ports and free one that is in use",
	Example: `  dev ports
  dev ports --all
  dev ports 3000
  dev ports 3000 --force --yes`,
	Args: cobra.MaximumNArgs(1),

	ValidArgsFunction: completePorts,

	// Failing to free a port isn't a usage mistake.
	SilenceUsage:  true,
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			if portsYes || portsForce {
				return errors.New("--yes and --force need a port")
			}

			return runList()
		}

		port, err := strconv.Atoi(args[0])
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("invalid port %q", args[0])
		}

		return runPort(port)
	},
}

func init() {
	Cmd.Flags().BoolVarP(
		&portsAll,
		"all",
		"a",
		false,
		"include ports outside your projects",
	)

	Cmd.Flags().BoolVar(
		&portsJSON,
		"json",
		false,
		"print ports as JSON without killing",
	)

	Cmd.Flags().BoolVarP(
		&portsYes,
		"yes",
		"y",
		false,
		"kill without confirmation",
	)

	Cmd.Flags().BoolVarP(
		&portsForce,
		"force",
		"f",
		false,
		"send SIGKILL instead of SIGTERM",
	)
}

func runList() error {
	projects, err := loadProjects()
	if err != nil {
		return err
	}

	listeners, err := ports.Scan(projects)
	if err != nil {
		return err
	}

	shown := make([]ports.Listener, 0, len(listeners))

	for _, l := range listeners {
		if portsAll || l.Project != "" {
			shown = append(shown, l)
		}
	}

	if portsJSON {
		return output.JSON(shown)
	}

	renderList(shown, len(listeners)-len(shown))

	return nil
}

func runPort(port int) error {
	// Freeing a port shouldn't depend on a valid config; without one, ports
	// just go unlabelled.
	projects, _ := loadProjects()

	listeners, err := ports.Scan(projects)
	if err != nil {
		return err
	}

	l, found := ports.Find(listeners, port)

	if portsJSON {
		result := []ports.Listener{}

		if found {
			result = append(result, l)
		}

		return output.JSON(result)
	}

	if !found {
		fmt.Println(
			output.Success.Render(
				fmt.Sprintf("Nothing is listening on port %d.", port),
			),
		)
		return nil
	}

	renderListener(l)

	if len(l.Processes) == 0 {
		if l.UID != os.Getuid() {
			return fmt.Errorf(
				"port %d is held by a process owned by %s; dev can only kill your own processes",
				port,
				l.User,
			)
		}

		return fmt.Errorf("can't find the process holding port %d", port)
	}

	if !portsYes {
		confirmed, err := confirmKill(l)
		if err != nil {
			return err
		}

		if !confirmed {
			return nil
		}
	}

	return killListener(l)
}

func loadProjects() ([]project.Project, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	return project.Discover(cfg.ProjectRoots)
}

func renderList(listeners []ports.Listener, hidden int) {
	subtitle := fmt.Sprintf("· %d in projects", len(listeners))

	if portsAll {
		subtitle = fmt.Sprintf("· %d listening", len(listeners))
	}

	fmt.Printf(
		"%s %s\n",
		output.Primary.Render("Ports"),
		output.Subtle.Render(subtitle),
	)

	fmt.Println(
		output.Rule.Render(strings.Repeat("─", 60)),
	)

	if len(listeners) == 0 {
		fmt.Println(
			output.Success.Render("No project ports listening."),
		)
	}

	for _, l := range listeners {
		fmt.Printf(
			"%s %s %s\n",
			output.PadRight(output.Primary.Render(strconv.Itoa(l.Port)), 7),
			output.PadRight(renderOwner(l), 18),
			renderProcesses(l),
		)
	}

	if hidden > 0 {
		fmt.Println()
		fmt.Println(
			output.Subtle.Render(
				fmt.Sprintf("%d more outside projects · show with --all", hidden),
			),
		)
	}
}

// renderOwner shows the listener's project, or the directory it runs in when
// that is outside every project.
func renderOwner(l ports.Listener) string {
	if l.Project != "" {
		return output.Text.Copy().Bold(true).Render(l.Project)
	}

	if len(l.Processes) > 0 && l.Processes[0].Dir != "" {
		return output.Subtle.Render(
			truncate(output.ShortenPath(l.Processes[0].Dir), 17),
		)
	}

	return output.Subtle.Render("—")
}

// renderProcesses shows the first process holding the port, and how many
// more there are.
func renderProcesses(l ports.Listener) string {
	if len(l.Processes) == 0 {
		return output.Subtle.Render("owned by " + l.User)
	}

	p := l.Processes[0]

	cell := output.PadRight(output.Subtle.Render(strconv.Itoa(p.PID)), 8) +
		" " +
		output.Muted.Render(truncate(commandLine(p), 36))

	if more := len(l.Processes) - 1; more > 0 {
		cell += output.Subtle.Render(fmt.Sprintf(" +%d more", more))
	}

	return cell
}

func renderListener(l ports.Listener) {
	fmt.Print(output.Primary.Render(fmt.Sprintf("Port %d", l.Port)))

	if l.Project != "" {
		fmt.Print(" " + output.Subtle.Render("· "+l.Project))
	}

	fmt.Println()

	fmt.Println(
		output.Rule.Render(strings.Repeat("─", 60)),
	)

	addresses := make([]string, 0, len(l.Addresses))

	for _, addr := range l.Addresses {
		addresses = append(addresses, addr.String())
	}

	printRow("Address", strings.Join(addresses, ", "))

	if len(l.Processes) == 0 {
		printRow("Owner", l.User)
		return
	}

	for _, p := range l.Processes {
		fmt.Println()
		printRow("Process", fmt.Sprintf("%s · pid %d", p.Name, p.PID))
		printRow("Command", truncate(commandLine(p), 100))

		if p.Dir != "" {
			printRow("Directory", output.ShortenPath(p.Dir))
		}
	}
}

func printRow(label, value string) {
	fmt.Printf(
		"%s %s\n",
		output.PadRight(
			output.Subtle.Render(label),
			14,
		),
		output.Text.Render(value),
	)
}

func confirmKill(l ports.Listener) (bool, error) {
	target := fmt.Sprintf("%d processes", len(l.Processes))

	if len(l.Processes) == 1 {
		p := l.Processes[0]
		target = fmt.Sprintf("%s (pid %d)", p.Name, p.PID)
	}

	verb := "Kill"
	if portsForce {
		verb = "Force kill"
	}

	fmt.Printf(
		"\n%s %s? %s ",
		verb,
		target,
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

// killListener kills every process holding l and waits for the port to be
// released.
func killListener(l ports.Listener) error {
	fmt.Println()

	failed := 0

	for _, p := range l.Processes {
		if err := ports.Kill(p, portsForce); err != nil {
			fmt.Printf("%s %s\n", output.Error.Render("✗"), err)
			failed++
		}
	}

	if failed == len(l.Processes) {
		return fmt.Errorf("port %d is still in use", l.Port)
	}

	closed, err := ports.WaitClosed(l, killTimeout)
	if err != nil {
		return err
	}

	if !closed {
		if portsForce {
			return fmt.Errorf("port %d is still in use after %s", l.Port, killTimeout)
		}

		return fmt.Errorf(
			"port %d is still in use after %s; retry with --force",
			l.Port,
			killTimeout,
		)
	}

	fmt.Printf(
		"%s Port %d is free\n",
		output.Success.Render("✓"),
		l.Port,
	)

	return nil
}

// commandLine formats a process's arguments for display: the program by its
// base name, and paths inside the working directory made relative.
func commandLine(p ports.Process) string {
	if len(p.Command) == 0 {
		return p.Name
	}

	args := make([]string, len(p.Command))
	args[0] = filepath.Base(p.Command[0])

	for i, arg := range p.Command[1:] {
		if p.Dir != "" {
			if rel, ok := strings.CutPrefix(arg, p.Dir+"/"); ok {
				arg = rel
			}
		}

		args[i+1] = arg
	}

	return strings.Join(args, " ")
}

func truncate(value string, width int) string {
	runes := []rune(value)

	if len(runes) <= width {
		return value
	}

	return string(runes[:width-1]) + "…"
}

// completePorts offers the ports dev can kill, described by their project or
// process name.
func completePorts(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]cobra.Completion, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	projects, _ := loadProjects()

	listeners, err := ports.Scan(projects)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	completions := make([]cobra.Completion, 0, len(listeners))

	for _, l := range listeners {
		if len(l.Processes) == 0 {
			continue
		}

		description := l.Processes[0].Name

		if l.Project != "" {
			description = l.Project + " · " + description
		}

		completions = append(
			completions,
			cobra.CompletionWithDesc(strconv.Itoa(l.Port), description),
		)
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
