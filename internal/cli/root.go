// Package cli wires forge's commands together.
package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"

	"github.com/madebyjonny/forge/internal/project"
	"github.com/madebyjonny/forge/internal/runner"
	"github.com/madebyjonny/forge/internal/scaffold"
	"github.com/madebyjonny/forge/internal/ui"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:           "forge",
	Short:         "Artisan-style CLI for Bun + Hono + Drizzle backends",
	Long:          ui.Logo() + "\n\nScaffold and manage Bun backends the Laravel way. Run with no arguments\ninside a project for an interactive command palette.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       scaffold.Version,
}

// exitError carries a child process exit code without printing a second error.
type exitError struct{ code int }

func (e exitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

func Execute() {
	if err := root.Execute(); err != nil {
		var ee exitError
		if errors.As(err, &ee) {
			os.Exit(ee.code)
		}
		ui.Error(err)
		os.Exit(1)
	}
}

func init() {
	root.RunE = runPalette
	root.PersistentFlags().BoolVarP(&ui.NoInteraction, "no-interaction", "n", false, "never prompt; use flags and defaults")
	root.AddGroup(
		&cobra.Group{ID: "project", Title: "Project:"},
		&cobra.Group{ID: "make", Title: "Generators:"},
		&cobra.Group{ID: "db", Title: "Database:"},
		&cobra.Group{ID: "run", Title: "Run:"},
		&cobra.Group{ID: "install", Title: "Features:"},
	)
	root.SetHelpCommandGroupID("project")
	root.SetCompletionCommandGroupID("project")
}

func add(group string, cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.GroupID = group
		root.AddCommand(c)
	}
}

// mustProject loads the project containing the working directory.
func mustProject() (*project.Project, error) {
	return project.Current()
}

// script runs a forge:* script, streaming output and propagating the exit code.
func script(p *project.Project, name string, o runner.Options) error {
	if err := runner.Script(p, name, o); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return exitError{runner.ExitCode(err)}
		}
		return err
	}
	return nil
}

func runPalette(cmd *cobra.Command, _ []string) error {
	p, err := project.Current()
	if err != nil || !ui.Interactive() {
		if err == nil {
			fmt.Println(ui.Faint.Render("project: ") + ui.Bold.Render(p.Manifest.Name) + "\n")
		}
		return cmd.Help()
	}

	var items []ui.PaletteItem
	for _, c := range root.Commands() {
		if c.Hidden || slices.Contains([]string{"help", "completion", "new"}, c.Name()) {
			continue
		}
		items = append(items, ui.PaletteItem{Name: c.Name(), Group: c.GroupID, Desc: c.Short})
	}
	slices.SortStableFunc(items, func(a, b ui.PaletteItem) int { return groupRank(a.Group) - groupRank(b.Group) })

	choice, err := ui.Palette("⚒ forge · "+p.Manifest.Name, items)
	if err != nil || choice == "" {
		return err
	}
	root.SetArgs([]string{choice})
	return root.Execute()
}

func groupRank(g string) int {
	return slices.Index([]string{"run", "make", "db", "install", "project"}, g)
}
