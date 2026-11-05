package cli

import (
	"fmt"
	"slices"

	"github.com/charmbracelet/huh"
	"github.com/madebyjonny/forge/internal/project"
	"github.com/madebyjonny/forge/internal/runner"
	"github.com/madebyjonny/forge/internal/scaffold"
	"github.com/madebyjonny/forge/internal/ui"
	"github.com/spf13/cobra"
)

func installFeatures(p *project.Project, names []string, skipInstall bool) error {
	resolved, err := scaffold.Resolve(names)
	if err != nil {
		return err
	}
	var todo []scaffold.Feature
	for _, f := range resolved {
		if p.Has(f.Name) {
			if slices.Contains(names, f.Name) {
				ui.Info("%s is already installed", f.Name)
			}
			continue
		}
		todo = append(todo, f)
	}
	if len(todo) == 0 {
		return nil
	}

	needsDeps, migrations := false, false
	steps := []ui.Step{}
	for _, f := range todo {
		f := f
		needsDeps = needsDeps || len(f.Deps)+len(f.DevDeps) > 0
		migrations = migrations || f.Migrations
		steps = append(steps, ui.Step{Title: "Installing " + f.Title, Run: func() error { return scaffold.Install(p, f) }})
	}
	if !skipInstall && runner.EnsureRuntime(p) == nil {
		if needsDeps {
			steps = append(steps, ui.Step{Title: "Installing dependencies", Run: func() error { return runner.Install(p) }})
		}
		if migrations {
			steps = append(steps,
				ui.Step{Title: "Generating migrations", Run: func() error { return runner.Quiet(p, "forge:migrate:generate", runner.Options{}) }},
				ui.Step{Title: "Migrating", Run: func() error { return runner.Quiet(p, "forge:migrate", runner.Options{}) }},
			)
		}
	}

	ui.Header("install")
	if err := ui.RunSteps(steps); err != nil {
		return err
	}
	var notes []string
	for _, f := range todo {
		notes = append(notes, f.Notes...)
	}
	fmt.Println()
	ui.Success("Installed %d feature(s)", len(todo))
	ui.Hint(notes...)
	return nil
}

func installCommands() []*cobra.Command {
	var skip bool
	all := &cobra.Command{
		Use:   "install [features...]",
		Short: "Add features to this project",
		Example: `  forge install                 # pick interactively
  forge install auth realtime`,
		ValidArgs: scaffold.FeatureNames(),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				if !ui.Interactive() {
					return fmt.Errorf("specify features: %v", scaffold.FeatureNames())
				}
				var opts []huh.Option[string]
				for _, f := range scaffold.Features {
					if !p.Has(f.Name) {
						opts = append(opts, huh.NewOption(f.Title+" — "+f.Description, f.Name))
					}
				}
				if len(opts) == 0 {
					ui.Info("Every feature is already installed")
					return nil
				}
				if err := huh.NewMultiSelect[string]().Title("Features to install").Options(opts...).Value(&args).Run(); err != nil {
					return err
				}
			}
			return installFeatures(p, args, skip)
		},
	}
	all.Flags().BoolVar(&skip, "no-install", false, "skip bun install and migrations")
	cmds := []*cobra.Command{all}

	for _, f := range scaffold.Features {
		name := f.Name
		var skipOne bool
		c := &cobra.Command{
			Use:   "install:" + name,
			Short: "Install " + f.Description,
			RunE: func(cmd *cobra.Command, _ []string) error {
				p, err := mustProject()
				if err != nil {
					return err
				}
				return installFeatures(p, []string{name}, skipOne)
			},
		}
		c.Flags().BoolVar(&skipOne, "no-install", false, "skip bun install and migrations")
		cmds = append(cmds, c)
	}
	return cmds
}

func aboutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "about",
		Short: "Show project information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			env, _ := project.OpenEnv(p.Path(".env"))
			get := func(k, d string) string {
				if v, ok := env.Get(k); ok && v != "" {
					return v
				}
				return d
			}
			bun := ui.Bad.Render("not installed")
			if runner.EnsureRuntime(p) == nil {
				bun = ui.OK.Render("installed")
			}
			pairs := [][2]string{
				{"Environment", ""},
				{"Name", p.Manifest.Name},
				{"Environment", get("NODE_ENV", "development")},
				{"URL", get("APP_URL", "http://localhost:3000")},
				{"Runtime", p.Manifest.Runtime + " (" + bun + ")"},
				{"forge", p.Manifest.Forge},
				{"Drivers", ""},
				{"Database", get("DB_CONNECTION", "sqlite")},
				{"Queue", get("QUEUE_CONNECTION", "sync")},
				{"Stack", ""},
			}
			keys := make([]string, 0, len(p.Manifest.Stack))
			for k := range p.Manifest.Stack {
				keys = append(keys, k)
			}
			slices.Sort(keys)
			for _, k := range keys {
				pairs = append(pairs, [2]string{k, p.Manifest.Stack[k]})
			}
			pairs = append(pairs, [2]string{"Features", ""})
			for _, f := range scaffold.Features {
				state := ui.Faint.Render("—")
				if p.Has(f.Name) {
					state = ui.OK.Render("installed")
				}
				pairs = append(pairs, [2]string{f.Name, state})
			}
			fmt.Println(ui.KV(pairs))
			return nil
		},
	}
}

func keyCmd() *cobra.Command {
	var show, force bool
	cmd := &cobra.Command{
		Use:   "key:generate",
		Short: "Set a new APP_KEY in .env",
		RunE: func(cmd *cobra.Command, _ []string) error {
			key := scaffold.Secret()
			if show {
				fmt.Println(key)
				return nil
			}
			p, err := mustProject()
			if err != nil {
				return err
			}
			env, err := project.OpenEnv(p.Path(".env"))
			if err != nil {
				return err
			}
			if v, ok := env.Get("APP_KEY"); ok && v != "" && !force {
				if !confirm("APP_KEY is already set. Replace it? (existing sessions/signatures become invalid)") {
					return fmt.Errorf("APP_KEY already set — pass --force to replace it")
				}
			}
			env.Set("APP_KEY", key)
			if err := env.Save(); err != nil {
				return err
			}
			ui.Success("Application key set")
			return nil
		},
	}
	cmd.Flags().BoolVar(&show, "show", false, "print a key instead of writing it")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "replace an existing key")
	return cmd
}

func init() {
	add("install", installCommands()...)
	add("project", aboutCmd(), keyCmd())
}
