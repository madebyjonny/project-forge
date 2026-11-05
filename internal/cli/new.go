package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/madebyjonny/forge/internal/project"
	"github.com/madebyjonny/forge/internal/runner"
	"github.com/madebyjonny/forge/internal/scaffold"
	"github.com/madebyjonny/forge/internal/ui"
	"github.com/spf13/cobra"
)

func newCmd() *cobra.Command {
	var (
		opts      scaffold.Options
		preset    string
		features  []string
		noInstall bool
		noGit     bool
	)
	cmd := &cobra.Command{
		Use:   "new [name]",
		Short: "Create a new project",
		Example: `  forge new                                  # interactive
  forge new blog                             # bare: hello-world routes on SQLite
  forge new blog --preset auth               # + sessions, OAuth, policies
  forge new blog --auth --realtime --db postgres
  forge new blog --preset full -n            # everything, no prompts`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				opts.Name = args[0]
			}
			for _, f := range scaffold.FeatureNames() {
				if on, _ := cmd.Flags().GetBool(f); on {
					features = append(features, f)
				}
			}
			configured := cmd.Flags().Changed("preset") || cmd.Flags().Changed("db") || len(features) > 0

			if ui.Interactive() && (opts.Name == "" || !configured) {
				if err := newWizard(&opts, &preset, &features); err != nil {
					return err
				}
			}
			if opts.Name == "" {
				return fmt.Errorf("a project name is required: forge new <name>")
			}
			if preset != "custom" {
				list, ok := scaffold.Presets[preset]
				if !ok {
					return fmt.Errorf("unknown preset %q (bare, auth, full)", preset)
				}
				features = append(features, list...)
			}
			slices.Sort(features)
			opts.Features = slices.Compact(features)
			if opts.Database != "sqlite" && opts.Database != "postgres" {
				return fmt.Errorf("--db must be sqlite or postgres")
			}
			return createProject(opts, !noInstall, !noGit)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&preset, "preset", "p", "bare", "bare | auth | full")
	f.StringVar(&opts.Database, "db", "sqlite", "sqlite | postgres")
	f.StringVar(&opts.DatabaseURL, "database-url", "", "postgres connection string")
	for _, feat := range scaffold.Features {
		f.Bool(feat.Name, false, "include "+strings.ToLower(feat.Title))
	}
	f.BoolVar(&noInstall, "no-install", false, "skip `bun install`")
	f.BoolVar(&noGit, "no-git", false, "skip git init")
	f.BoolVarP(&opts.Force, "force", "f", false, "write into a non-empty directory")
	return cmd
}

func newWizard(o *scaffold.Options, preset *string, features *[]string) error {
	fmt.Println()
	fmt.Println(" " + ui.Logo())
	fmt.Println()

	var groups []*huh.Group
	if o.Name == "" {
		groups = append(groups, huh.NewGroup(
			huh.NewInput().Title("Project name").Placeholder("my-app").Value(&o.Name).Validate(scaffold.ValidateName),
		))
	}
	groups = append(groups,
		huh.NewGroup(
			huh.NewSelect[string]().Title("Starter").Options(
				huh.NewOption("Bare — hello-world routes, ready to build on", "bare"),
				huh.NewOption("Auth — sessions, OAuth & policies", "auth"),
				huh.NewOption("Full — auth, realtime, queues & docker", "full"),
				huh.NewOption("Custom — pick features", "custom"),
			).Value(preset),
		),
		huh.NewGroup(
			huh.NewMultiSelect[string]().Title("Features").Options(featureOptions(*features)...).Value(features),
		).WithHideFunc(func() bool { return *preset != "custom" }),
		huh.NewGroup(
			huh.NewSelect[string]().Title("Database").Description("Switch any environment later with `forge db:use`").Options(
				huh.NewOption("SQLite — zero setup", "sqlite"),
				huh.NewOption("PostgreSQL", "postgres"),
			).Value(&o.Database),
		),
		huh.NewGroup(
			huh.NewInput().Title("Postgres URL").Placeholder("postgres://postgres:postgres@localhost:5432/app").Value(&o.DatabaseURL),
		).WithHideFunc(func() bool { return o.Database != "postgres" }),
	)
	return huh.NewForm(groups...).WithTheme(huh.ThemeCharm()).Run()
}

func featureOptions(selected []string) []huh.Option[string] {
	var opts []huh.Option[string]
	for _, f := range scaffold.Features {
		opts = append(opts, huh.NewOption(f.Title+" — "+f.Description, f.Name).Selected(slices.Contains(selected, f.Name)))
	}
	return opts
}

func createProject(o scaffold.Options, install, git bool) error {
	var p *project.Project
	hasBun := runner.EnsureRuntime(nil) == nil
	migrations := false
	for _, name := range o.Features {
		if f, _ := scaffold.GetFeature(name); f.Migrations {
			migrations = true
		}
	}

	ui.Header("new " + o.Name)
	steps := []ui.Step{{Title: "Creating project files", Run: func() (err error) {
		p, err = scaffold.Create(o)
		return err
	}}}
	if install && hasBun {
		steps = append(steps, ui.Step{Title: "Installing dependencies", Run: func() error { return runner.Install(p) }})
		if migrations {
			steps = append(steps, ui.Step{Title: "Generating migrations", Run: func() error {
				return runner.Quiet(p, "forge:migrate:generate", runner.Options{})
			}})
		}
		if o.Database == "sqlite" {
			steps = append(steps, ui.Step{Title: "Migrating SQLite database", Run: func() error {
				return runner.Quiet(p, "forge:migrate", runner.Options{})
			}})
		}
	}
	if _, err := exec.LookPath("git"); err == nil && git {
		steps = append(steps, ui.Step{Title: "Initialising git", Run: func() error { return gitInit(p.Root) }})
	}
	if err := ui.RunSteps(steps); err != nil {
		return err
	}

	fmt.Println()
	ui.Success("Created %s %s", ui.Bold.Render(o.Name), ui.Faint.Render(featureSummary(o)))
	if install && !hasBun {
		ui.Warn("Bun not found — install it from https://bun.sh, then run `bun install` and `forge migrate`")
	}
	next := []string{"cd " + relDir(p.Root), "forge serve"}
	if !install {
		next = slices.Insert(next, 1, "bun install", "forge make:migration && forge migrate")
	} else if o.Database == "postgres" {
		next = slices.Insert(next, 1, "forge migrate   "+ui.Faint.Render("# once Postgres is reachable"))
	}
	for i := range next {
		next[i] = ui.Code.Render(next[i])
	}
	for _, name := range o.Features {
		f, _ := scaffold.GetFeature(name)
		next = append(next, f.Notes...)
	}
	ui.Hint(next...)
	return nil
}

func featureSummary(o scaffold.Options) string {
	parts := []string{o.Database}
	parts = append(parts, o.Features...)
	return "(" + strings.Join(parts, ", ") + ")"
}

func relDir(dir string) string {
	wd, _ := os.Getwd()
	if rel, err := filepath.Rel(wd, dir); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return dir
}

func gitInit(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return nil
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			return fmt.Errorf("git %s: %s", args[0], out)
		}
	}
	c := exec.Command("git", "commit", "-q", "-m", "Initial commit (forge)")
	c.Dir = dir
	_ = c.Run() // fine to skip if git identity isn't configured
	return nil
}

func init() { add("project", newCmd()) }
