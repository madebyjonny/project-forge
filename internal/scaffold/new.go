package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/madebyjonny/forge/internal/project"
)

type Options struct {
	Name        string
	Dir         string
	Database    string // sqlite | postgres
	DatabaseURL string
	Features    []string
	Force       bool
}

// Presets bundle features, like `laravel new --breeze`.
var Presets = map[string][]string{
	"bare": {},
	"auth": {"auth"},
	"full": {"auth", "mcp", "realtime", "queue", "docker"},
}

var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

func ValidateName(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("project name %q must be lowercase letters, numbers, '-', '_' or '.'", name)
	}
	return nil
}

// BaseScripts is the CLI ↔ project contract: forge commands only ever call these
// scripts, so swapping the ORM or runtime means editing scripts, not commands.
var BaseScripts = map[string]string{
	"dev":                    "bun --hot server.ts",
	"start":                  "bun server.ts",
	"test":                   "bun test",
	"forge:dev":              "bun --hot server.ts",
	"forge:start":            "bun server.ts",
	"forge:test":             "bun test",
	"forge:typecheck":        "tsc --noEmit",
	"forge:migrate":          "bun core/console/migrate.ts",
	"forge:migrate:generate": "bun core/console/generate.ts",
	"forge:seed":             "bun core/console/seed.ts",
	"forge:studio":           "bun --bun x drizzle-kit studio",
	"forge:routes":           "bun core/console/routes.ts",
	"forge:queue:work":       "bun core/console/work.ts",
}

// Create writes a new project to disk. Feature installs happen here too, but
// dependency installation and migrations are separate steps.
func Create(o Options) (*project.Project, error) {
	if err := ValidateName(o.Name); err != nil {
		return nil, err
	}
	if o.Dir == "" {
		o.Dir = o.Name
	}
	dir, err := filepath.Abs(o.Dir)
	if err != nil {
		return nil, err
	}
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 && !o.Force {
		return nil, fmt.Errorf("%s already exists and is not empty (use --force)", dir)
	}
	if _, err := copyTree("base", dir, true); err != nil {
		return nil, err
	}

	pkg := project.NewPackageJSON(filepath.Join(dir, "package.json"), map[string]any{
		"name":    o.Name,
		"private": true,
		"type":    "module",
	})
	pkg.SetScripts(BaseScripts)
	pkg.AddDeps(deps("hono", "drizzle-orm", "zod", "pg"), false)
	pkg.AddDeps(deps("drizzle-kit", "@types/bun", "@types/pg", "typescript"), true)
	if err := pkg.Save(); err != nil {
		return nil, err
	}

	if err := writeEnv(dir, o); err != nil {
		return nil, err
	}

	p := &project.Project{Root: dir, Manifest: project.Manifest{
		Name:    o.Name,
		Forge:   Version,
		Runtime: "bun",
		Stack: map[string]string{
			"http":       "hono",
			"orm":        "drizzle",
			"validation": "zod",
		},
		Features: []string{},
	}}
	if err := p.Save(); err != nil {
		return nil, err
	}

	features, err := Resolve(o.Features)
	if err != nil {
		return nil, err
	}
	for _, f := range features {
		if err := Install(p, f); err != nil {
			return nil, fmt.Errorf("install %s: %w", f.Name, err)
		}
	}
	if err := WriteBootstrap(p); err != nil {
		return nil, err
	}
	if err := writeReadme(p); err != nil {
		return nil, err
	}
	return p, nil
}

func writeEnv(dir string, o Options) error {
	pgURL := o.DatabaseURL
	if pgURL == "" {
		pgURL = "postgres://postgres:postgres@localhost:5432/" + regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(o.Name, "_")
	}
	for _, file := range []string{".env", ".env.example"} {
		env, err := project.OpenEnv(filepath.Join(dir, file))
		if err != nil {
			return err
		}
		key := ""
		if file == ".env" {
			key = Secret()
		}
		env.Set("APP_NAME", o.Name)
		env.Set("NODE_ENV", "development")
		env.Set("APP_URL", "http://localhost:3000")
		env.Set("APP_KEY", key)
		env.Set("PORT", "3000")
		env.Section("Database — switch per environment, e.g. in .env.production (`forge db:use postgres`)")
		if o.Database == "postgres" {
			env.Set("DB_CONNECTION", "postgres")
			env.Set("DATABASE_URL", pgURL)
		} else {
			env.Set("DB_CONNECTION", "sqlite")
			env.Set("DB_DATABASE", "database/database.sqlite")
		}
		env.Section("Queue — sync runs jobs inline; redis uses workers (`forge install:queue`)")
		env.Set("QUEUE_CONNECTION", "sync")
		if err := env.Save(); err != nil {
			return err
		}
	}
	return nil
}
