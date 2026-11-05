package cli

import (
	"fmt"
	"regexp"

	"github.com/charmbracelet/huh"
	"github.com/madebyjonny/forge/internal/project"
	"github.com/madebyjonny/forge/internal/runner"
	"github.com/madebyjonny/forge/internal/ui"
	"github.com/spf13/cobra"
)

// envFlag adds --env, which selects .env.<env> via NODE_ENV (≈ artisan --env).
func envFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVarP(target, "env", "e", "", "environment, e.g. production (loads .env.production)")
}

func dbCommands() []*cobra.Command {
	var env string
	var seed, fresh, force bool
	migrate := &cobra.Command{
		Use:   "migrate",
		Short: "Run pending migrations",
		Example: `  forge migrate
  forge migrate --fresh --seed
  forge migrate --env production`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			if fresh && env == "production" && !force && !confirm("Drop ALL tables in production?") {
				return fmt.Errorf("aborted")
			}
			var extra []string
			if fresh {
				extra = append(extra, "--fresh")
			}
			if seed {
				extra = append(extra, "--seed")
			}
			return script(p, "forge:migrate", runner.Options{Env: env, Extra: extra})
		},
	}
	envFlag(migrate, &env)
	migrate.Flags().BoolVar(&fresh, "fresh", false, "drop all tables first")
	migrate.Flags().BoolVar(&seed, "seed", false, "run seeders afterwards")
	migrate.Flags().BoolVarP(&force, "force", "f", false, "allow --fresh in production without confirmation")

	var freshEnv string
	var freshSeed, freshForce bool
	migrateFresh := &cobra.Command{
		Use:   "migrate:fresh",
		Short: "Drop all tables and re-run migrations",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			if freshEnv == "production" && !freshForce && !confirm("Drop ALL tables in production?") {
				return fmt.Errorf("aborted")
			}
			extra := []string{"--fresh"}
			if freshSeed {
				extra = append(extra, "--seed")
			}
			return script(p, "forge:migrate", runner.Options{Env: freshEnv, Extra: extra})
		},
	}
	envFlag(migrateFresh, &freshEnv)
	migrateFresh.Flags().BoolVar(&freshSeed, "seed", false, "run seeders afterwards")
	migrateFresh.Flags().BoolVarP(&freshForce, "force", "f", false, "run in production without confirmation")

	makeMigration := &cobra.Command{
		Use:     "make:migration [name]",
		Aliases: []string{"migrate:generate"},
		Short:   "Generate migrations from model changes (SQLite + Postgres)",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			var extra []string
			if len(args) == 1 {
				extra = []string{"--name", regexp.MustCompile(`[^a-z0-9_]+`).ReplaceAllString(args[0], "_")}
			}
			return script(p, "forge:migrate:generate", runner.Options{Extra: extra})
		},
	}

	var seedEnv string
	dbSeed := &cobra.Command{
		Use:   "db:seed",
		Short: "Run database seeders",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			return script(p, "forge:seed", runner.Options{Env: seedEnv})
		},
	}
	envFlag(dbSeed, &seedEnv)

	studio := &cobra.Command{
		Use:   "db:studio",
		Short: "Open a database browser",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			return script(p, "forge:studio", runner.Options{})
		},
	}

	var useEnv, url, file string
	use := &cobra.Command{
		Use:   "db:use [sqlite|postgres]",
		Short: "Switch the database driver for an environment",
		Example: `  forge db:use postgres --url postgres://user:pass@localhost:5432/app
  forge db:use postgres --env production --url $DATABASE_URL
  forge db:use sqlite`,
		Args:      cobra.MaximumNArgs(1),
		ValidArgs: []string{"sqlite", "postgres"},
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			driver := ""
			if len(args) == 1 {
				driver = args[0]
			} else if ui.Interactive() {
				err := huh.NewSelect[string]().Title("Database driver").
					Options(huh.NewOption("SQLite", "sqlite"), huh.NewOption("PostgreSQL", "postgres")).
					Value(&driver).Run()
				if err != nil {
					return err
				}
			}
			return useDatabase(p, driver, useEnv, url, file)
		},
	}
	envFlag(use, &useEnv)
	use.Flags().StringVar(&url, "url", "", "Postgres connection URL")
	use.Flags().StringVar(&file, "file", "", "SQLite file (default database/database.sqlite)")

	return []*cobra.Command{migrate, migrateFresh, makeMigration, dbSeed, studio, use}
}

func useDatabase(p *project.Project, driver, env, url, file string) error {
	envFile := ".env"
	if env != "" && env != "development" {
		envFile = ".env." + env
	}
	e, err := project.OpenEnv(p.Path(envFile))
	if err != nil {
		return err
	}
	switch driver {
	case "postgres", "pg", "postgresql":
		if url == "" {
			url, _ = e.Get("DATABASE_URL")
		}
		if url == "" && ui.Interactive() {
			url = "postgres://postgres:postgres@localhost:5432/" + p.Manifest.Name
			if err := huh.NewInput().Title("Postgres URL").Value(&url).Run(); err != nil {
				return err
			}
		}
		if url == "" {
			return fmt.Errorf("--url is required for postgres")
		}
		e.Set("DB_CONNECTION", "postgres")
		e.Set("DATABASE_URL", url)
	case "sqlite":
		e.Set("DB_CONNECTION", "sqlite")
		if file != "" {
			e.Set("DB_DATABASE", file)
		} else {
			e.Default("DB_DATABASE", "database/database.sqlite")
		}
	default:
		return fmt.Errorf("unknown driver %q (sqlite, postgres)", driver)
	}
	if err := e.Save(); err != nil {
		return err
	}
	ui.Success("%s now uses %s", ui.Bold.Render(envFile), ui.Accent.Render(driver))
	hint := "forge migrate"
	if env != "" {
		hint += " --env " + env
	}
	ui.Hint("Apply the schema: " + ui.Code.Render(hint))
	return nil
}

func confirm(q string) bool {
	if !ui.Interactive() {
		return false
	}
	ok := false
	_ = huh.NewConfirm().Title(q).Value(&ok).Run()
	return ok
}

func init() { add("db", dbCommands()...) }
