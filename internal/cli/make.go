package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/madebyjonny/forge/internal/gen"
	"github.com/madebyjonny/forge/internal/project"
	"github.com/madebyjonny/forge/internal/runner"
	"github.com/madebyjonny/forge/internal/ui"
	"github.com/spf13/cobra"
)

// askName returns args[0] or prompts for it.
func askName(args []string, title, placeholder string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}
	if !ui.Interactive() {
		return "", fmt.Errorf("missing name argument")
	}
	var name string
	err := huh.NewInput().Title(title).Placeholder(placeholder).Value(&name).
		Validate(func(s string) error {
			if gen.Pascal(s) == "" {
				return fmt.Errorf("required")
			}
			return nil
		}).Run()
	return name, err
}

// askFields prompts for "name:type" specs when none were given.
func askFields(specs []string) ([]gen.Field, error) {
	if len(specs) == 0 && ui.Interactive() {
		var raw string
		err := huh.NewInput().
			Title("Fields").
			Description("space separated name:type[:modifier] — " + gen.FieldTypes + "\nmodifiers: nullable, unique, default=value · e.g. title:string body:text? user:references").
			Placeholder("title:string body:text published:boolean:default=false").
			Value(&raw).
			Validate(func(s string) error { _, err := gen.ParseFields(strings.Fields(s)); return err }).
			Run()
		if err != nil {
			return nil, err
		}
		specs = strings.Fields(raw)
	}
	return gen.ParseFields(specs)
}

func report(g *gen.Generator) {
	fmt.Println()
	for _, f := range g.Created {
		ui.File("created", f)
	}
	for _, f := range g.Updated {
		ui.File("updated", f)
	}
	for _, n := range g.Notes {
		ui.Warn("%s", n)
	}
	fmt.Println()
}

// generateMigrations runs make:migration (+ migrate) after schema changes.
func generateMigrations(p *project.Project, migrate bool) error {
	steps := []ui.Step{{Title: "Generating migrations", Run: func() error {
		return runner.Quiet(p, "forge:migrate:generate", runner.Options{})
	}}}
	if migrate {
		steps = append(steps, ui.Step{Title: "Migrating", Run: func() error { return runner.Quiet(p, "forge:migrate", runner.Options{}) }})
	}
	return ui.RunSteps(steps)
}

type makeSpec struct {
	use, short, example, prompt, placeholder string
	run                                      func(g *gen.Generator, name string, args []string, cmd *cobra.Command) error
	flags                                    func(cmd *cobra.Command)
}

func makeCmd(s makeSpec) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     s.use,
		Short:   s.short,
		Example: s.example,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			name, err := askName(args, s.prompt, s.placeholder)
			if err != nil {
				return err
			}
			rest := []string{}
			if len(args) > 1 {
				rest = args[1:]
			}
			g := gen.New(p, force)
			runErr := s.run(g, name, rest, cmd)
			report(g)
			if runErr == nil && g.Migrate {
				return generateMigrations(p, true)
			}
			return runErr
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing files")
	if s.flags != nil {
		s.flags(cmd)
	}
	return cmd
}

func boolFlag(cmd *cobra.Command, name string) bool {
	v, _ := cmd.Flags().GetBool(name)
	return v
}

func init() {
	add("make",
		makeCmd(makeSpec{
			use:   "make:model [name] [fields...]",
			short: "Create a model (table + query helpers)",
			example: `  forge make:model Post title:string body:text? published:boolean:default=false
  forge make:model Comment body:text post:references user:references -m
  forge make:model Post title:string -a     # + request, controller, policy, routes, test`,
			prompt: "Model name", placeholder: "Post",
			flags: func(c *cobra.Command) {
				c.Flags().BoolP("migration", "m", false, "generate and run migrations")
				c.Flags().BoolP("controller", "c", false, "also create a resource controller + routes")
				c.Flags().BoolP("request", "r", false, "also create a request validator")
				c.Flags().BoolP("all", "a", false, "model, request, controller, policy, routes, test + migrate")
			},
			run: func(g *gen.Generator, name string, args []string, cmd *cobra.Command) error {
				fields, err := askFields(args)
				if err != nil {
					return err
				}
				if boolFlag(cmd, "all") {
					if err := g.Resource(name, fields); err != nil {
						return err
					}
					g.Migrate = true
					return nil
				}
				if err := g.Model(name, fields); err != nil {
					return err
				}
				if boolFlag(cmd, "request") || boolFlag(cmd, "controller") {
					if err := g.Validator(name, fields); err != nil {
						return err
					}
				}
				if boolFlag(cmd, "controller") {
					if err := g.Controller(name, true); err != nil {
						return err
					}
					if err := g.Routes(name); err != nil {
						return err
					}
				}
				if boolFlag(cmd, "migration") {
					g.Migrate = true
					return nil
				}
				g.Notes = append(g.Notes, "Run `forge make:migration && forge migrate` to create the table")
				return nil
			},
		}),
		makeCmd(makeSpec{
			use:   "make:resource [name] [fields...]",
			short: "Model + request + controller + policy + routes + test, migrated",
			example: `  forge make:resource Post title:string body:text published:boolean:default=false user:references
  forge make:resource Product name:string price:float --no-migrate`,
			prompt: "Resource name", placeholder: "Post",
			flags: func(c *cobra.Command) {
				c.Flags().Bool("no-migrate", false, "don't generate/run migrations")
				c.Flags().Bool("mcp", false, "also expose CRUD tools to MCP clients")
			},
			run: func(g *gen.Generator, name string, args []string, cmd *cobra.Command) error {
				fields, err := askFields(args)
				if err != nil {
					return err
				}
				if err := g.Resource(name, fields); err != nil {
					return err
				}
				if boolFlag(cmd, "mcp") {
					if err := g.Tool(name, true); err != nil {
						return err
					}
				}
				if boolFlag(cmd, "no-migrate") {
					return nil
				}
				g.Migrate = true
				return nil
			},
		}),
		makeCmd(makeSpec{
			use:     "make:controller [name]",
			short:   "Create a controller",
			example: "  forge make:controller Dashboard\n  forge make:controller Post --resource --routes",
			prompt:  "Controller name", placeholder: "PostController",
			flags: func(c *cobra.Command) {
				c.Flags().BoolP("resource", "r", false, "CRUD actions backed by the model of the same name")
				c.Flags().Bool("routes", false, "register resource routes in routes/api.ts")
			},
			run: func(g *gen.Generator, name string, _ []string, cmd *cobra.Command) error {
				if err := g.Controller(name, boolFlag(cmd, "resource")); err != nil {
					return err
				}
				if boolFlag(cmd, "routes") {
					return g.Routes(name)
				}
				return nil
			},
		}),
		makeCmd(makeSpec{
			use:     "make:request [name] [fields...]",
			short:   "Create a request validator (Zod)",
			example: "  forge make:request Post title:string body:text",
			prompt:  "Request name", placeholder: "Post",
			run: func(g *gen.Generator, name string, args []string, _ *cobra.Command) error {
				fields, err := gen.ParseFields(args)
				if err != nil {
					return err
				}
				return g.Validator(name, fields)
			},
		}),
		makeCmd(makeSpec{
			use:    "make:policy [name]",
			short:  "Create an authorization policy",
			prompt: "Policy for model", placeholder: "Post",
			run: func(g *gen.Generator, name string, _ []string, _ *cobra.Command) error { return g.Policy(name) },
		}),
		makeCmd(makeSpec{
			use:   "make:tool [name]",
			short: "Create an MCP tool",
			example: `  forge make:tool SendReport
  forge make:tool Post --resource      # list/get/create/update/delete tools, policy-checked`,
			prompt: "Tool name", placeholder: "SendReport",
			flags: func(c *cobra.Command) {
				c.Flags().BoolP("resource", "r", false, "CRUD tools for the model of the same name")
			},
			run: func(g *gen.Generator, name string, _ []string, cmd *cobra.Command) error {
				return g.Tool(name, boolFlag(cmd, "resource"))
			},
		}),
		makeCmd(makeSpec{
			use:    "make:middleware [name]",
			short:  "Create a named middleware",
			prompt: "Middleware name", placeholder: "EnsureTeamMember",
			run: func(g *gen.Generator, name string, _ []string, _ *cobra.Command) error { return g.Middleware(name) },
		}),
		makeCmd(makeSpec{
			use:    "make:job [name]",
			short:  "Create a queued job",
			prompt: "Job name", placeholder: "SendWelcomeEmail",
			run: func(g *gen.Generator, name string, _ []string, _ *cobra.Command) error { return g.Job(name) },
		}),
		makeCmd(makeSpec{
			use:    "make:seeder [name]",
			short:  "Create a database seeder",
			prompt: "Seeder name", placeholder: "UserSeeder",
			run: func(g *gen.Generator, name string, _ []string, _ *cobra.Command) error { return g.Seeder(name) },
		}),
		makeCmd(makeSpec{
			use:    "make:test [name]",
			short:  "Create a test",
			prompt: "Test name", placeholder: "Checkout",
			run: func(g *gen.Generator, name string, _ []string, _ *cobra.Command) error { return g.Test(name, false) },
		}),
	)

	var force bool
	publish := &cobra.Command{
		Use:   "stub:publish",
		Short: "Copy generator stubs into ./stubs to customise them",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			g := gen.New(p, force)
			err = g.PublishStubs()
			report(g)
			return err
		},
	}
	publish.Flags().BoolVarP(&force, "force", "f", false, "overwrite existing stubs")
	add("make", publish)
}
