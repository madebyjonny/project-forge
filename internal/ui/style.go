// Package ui holds forge's terminal styling and Bubble Tea components.
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

var (
	Brand  = lipgloss.AdaptiveColor{Light: "#D9480F", Dark: "#FF8A3D"}
	Green  = lipgloss.AdaptiveColor{Light: "#2B8A3E", Dark: "#69DB7C"}
	Red    = lipgloss.AdaptiveColor{Light: "#C92A2A", Dark: "#FF6B6B"}
	Yellow = lipgloss.AdaptiveColor{Light: "#E67700", Dark: "#FFD43B"}
	Blue   = lipgloss.AdaptiveColor{Light: "#1864AB", Dark: "#74C0FC"}
	Muted  = lipgloss.AdaptiveColor{Light: "#868E96", Dark: "#6C757D"}

	Bold      = lipgloss.NewStyle().Bold(true)
	Faint     = lipgloss.NewStyle().Foreground(Muted)
	Accent    = lipgloss.NewStyle().Foreground(Brand).Bold(true)
	OK        = lipgloss.NewStyle().Foreground(Green)
	Bad       = lipgloss.NewStyle().Foreground(Red)
	Warning   = lipgloss.NewStyle().Foreground(Yellow)
	Code      = lipgloss.NewStyle().Foreground(Blue)
	Badge     = lipgloss.NewStyle().Foreground(lipgloss.Color("#1A1A1A")).Background(Brand).Bold(true).Padding(0, 1)
	ErrBadge  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Background(Red).Bold(true).Padding(0, 1)
	InfoBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#1A1A1A")).Background(Blue).Bold(true).Padding(0, 1)
)

// NoInteraction disables prompts and animated output (--no-interaction / CI).
var NoInteraction bool

// Interactive reports whether we can show prompts and animations.
func Interactive() bool {
	if NoInteraction || os.Getenv("CI") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func Logo() string {
	return Accent.Render("⚒ forge") + Faint.Render("  artisan for Bun · Hono · Drizzle")
}

func Header(title string) {
	fmt.Println()
	fmt.Println(" " + Badge.Render(strings.ToUpper(title)))
	fmt.Println()
}

func Success(format string, a ...any) {
	fmt.Println(" " + OK.Render("✓") + " " + fmt.Sprintf(format, a...))
}

func Info(format string, a ...any) {
	fmt.Println(" " + InfoBadge.Render("INFO") + " " + fmt.Sprintf(format, a...))
}

func Warn(format string, a ...any) {
	fmt.Println(" " + Warning.Render("!") + " " + fmt.Sprintf(format, a...))
}

func Error(err error) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, " "+ErrBadge.Render("ERROR")+" "+err.Error())
	fmt.Fprintln(os.Stderr)
}

// File prints a created/updated file line.
func File(verb, path string) {
	style := OK
	if verb == "updated" {
		style = Warning
	}
	fmt.Printf("  %s %s\n", style.Render(fmt.Sprintf("%-8s", verb)), path)
}

func Hint(lines ...string) {
	fmt.Println()
	for _, l := range lines {
		fmt.Println("  " + Faint.Render("›") + " " + l)
	}
	fmt.Println()
}

// Table renders rows with a header in forge's style.
func Table(headers []string, rows [][]string) string {
	return table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(Faint).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().Padding(0, 1)
			if row == table.HeaderRow {
				return s.Foreground(Brand).Bold(true)
			}
			return s
		}).
		Render()
}

// KV renders aligned key/value lines (used by `forge about`).
func KV(pairs [][2]string) string {
	width := 0
	for _, p := range pairs {
		width = max(width, lipgloss.Width(p[0]))
	}
	var b strings.Builder
	for _, p := range pairs {
		if p[1] == "" && p[0] != "" {
			b.WriteString("\n " + Accent.Render(p[0]) + "\n")
			continue
		}
		dots := strings.Repeat(".", max(2, width+6-lipgloss.Width(p[0])))
		b.WriteString("  " + p[0] + " " + Faint.Render(dots) + " " + p[1] + "\n")
	}
	return b.String()
}
