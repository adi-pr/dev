package project

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/adi-pr/dev/internal/output"
)

func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"

	case duration < time.Hour:
		minutes := int(duration.Minutes())

		if minutes == 1 {
			return "1 minute ago"
		}

		return fmt.Sprintf("%d minutes ago", minutes)

	case duration < 24*time.Hour:
		hours := int(duration.Hours())

		if hours == 1 {
			return "1 hour ago"
		}

		return fmt.Sprintf("%d hours ago", hours)

	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)

		if days == 1 {
			return "1 day ago"
		}

		return fmt.Sprintf("%d days ago", days)

	default:
		return t.Format("2006-01-02")
	}
}

func renderState(dirty bool) string {
	if dirty {
		return output.Warning.Render("● dirty")
	}

	return output.Success.Render("● clean")
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	return strings.Replace(path, home, "~", 1)
}
