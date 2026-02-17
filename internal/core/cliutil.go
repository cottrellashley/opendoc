package core

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ── CLI colour styles ───────────────────────────────────────

var (
	CLIAccent  = lipgloss.NewStyle().Foreground(lipgloss.Color("#79a5f2"))
	CLISuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("#3fb950"))
	CLIError   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f85149"))
	CLIWarn    = lipgloss.NewStyle().Foreground(lipgloss.Color("#d29922"))
	CLIMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7d8590"))
	CLIBold    = lipgloss.NewStyle().Bold(true)
)

// ── Prefixed output helpers ─────────────────────────────────

func InfoMsg(msg string)    { fmt.Println(CLIAccent.Render("  info") + "  " + msg) }
func OkMsg(msg string)      { fmt.Println(CLISuccess.Render("    ok") + "  " + msg) }
func ErrMsg(msg string)     { fmt.Println(CLIError.Render("   err") + "  " + msg) }
func WarnMsg(msg string)    { fmt.Println(CLIWarn.Render("  warn") + "  " + msg) }
func StepMsg(msg string)    { fmt.Println(CLIMuted.Render("     ·") + "  " + msg) }
func DoneMsg(msg string)    { fmt.Println(CLISuccess.Render("  done") + "  " + msg) }

// StatusLine returns a labelled value line for status output.
func StatusLine(label, value string) string {
	return fmt.Sprintf("  %s  %s", CLIMuted.Render(fmt.Sprintf("%16s", label)), value)
}

// Banner prints the OpenDoc CLI header.
func Banner() {
	fmt.Println()
	fmt.Println(CLIAccent.Bold(true).Render("  opendoc") + CLIMuted.Render(" v2.0.0"))
	fmt.Println()
}

// TimeSince returns a human-readable "time ago" string.
func TimeSince(ms int64) string {
	t := time.UnixMilli(ms)
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return t.Format("2006-01-02 15:04")
	}
}
