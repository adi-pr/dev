package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/adi-pr/dev/internal/config"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the config file in your editor",
	Args:  cobra.NoArgs,

	// An editor exiting non-zero isn't a usage mistake.
	SilenceUsage:  true,
	SilenceErrors: true,

	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.Path()
		if err != nil {
			return err
		}

		// A missing file is fine: the editor creates it on save.
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("create config directory: %w", err)
		}

		editor := strings.Fields(editorCommand())

		command := exec.Command(editor[0], append(editor[1:], path)...)

		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Stdin = os.Stdin

		// Wait, unlike project open: a terminal editor needs the TTY until
		// it exits.
		if err := command.Run(); err != nil {
			return fmt.Errorf("%s: %w", editor[0], err)
		}

		return nil
	},
}

// editorCommand prefers $EDITOR, then the configured editor, then the
// default. $EDITOR may carry arguments, e.g. "code --wait". A broken config
// is skipped rather than reported, since editing is how it gets fixed.
func editorCommand() string {
	if editor := os.Getenv("EDITOR"); strings.TrimSpace(editor) != "" {
		return editor
	}

	if cfg, err := config.Load(); err == nil && cfg.Editor != "" {
		return cfg.Editor
	}

	return config.DefaultEditor
}
