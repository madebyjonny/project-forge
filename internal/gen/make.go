package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/madebyjonny/forge/internal/project"
	"github.com/madebyjonny/forge/internal/scaffold"
)

// Generator writes files into a project. Every method records what it touched.
type Generator struct {
	P       *project.Project
	Force   bool
	Created []string
	Updated []string
	Notes   []string
	// Migrate asks the caller to generate + run migrations after reporting.
	Migrate bool
}

func New(p *project.Project, force bool) *Generator { return &Generator{P: p, Force: force} }

func (g *Generator) write(rel, stub string, data any) error {
	content, err := scaffold.Stub(g.P.Root, stub, data)
	if err != nil {
		return err
	}
	if err := project.WriteFile(g.P.Path(rel), content, g.Force); err != nil {
		return err
	}
	g.Created = append(g.Created, rel)
	return nil
}

func (g *Generator) register(rel, marker, line string) error {
	if err := project.InsertBefore(g.P.Path(rel), marker, line); err != nil {
		return err
	}
	for _, u := range g.Updated {
		if u == rel {
			return nil
		}
	}
	g.Updated = append(g.Updated, rel)
	return nil
}

func (g *Generator) requireORM() error {
	if orm := g.P.Stack("orm"); orm != "drizzle" {
		return fmt.Errorf("no model generator for orm %q — add stubs/model.ts.tmpl to this project", orm)
	}
	return nil
}

// Model creates app/models/<name>.ts and registers it in the schema.
func (g *Generator) Model(name string, fields []Field) error {
	if err := g.requireORM(); err != nil {
		return err
	}
	n := NamesFor(name)
	var imports, columns []string
	seen := map[string]bool{}
	for _, f := range fields {
		columns = append(columns, f.ColumnCode())
		if f.Type != "references" {
			continue
		}
		ref := NamesFor(f.Ref)
		if ref.File == n.File {
			continue
		}
		if !g.P.Exists("app/models/" + ref.File + ".ts") {
			g.Notes = append(g.Notes, fmt.Sprintf("%s references %s, but app/models/%s.ts doesn't exist yet", n.Name, f.Ref, ref.File))
		}
		if imp := fmt.Sprintf("import { %s } from %q;", ref.Var, "./"+ref.File); !seen[imp] {
			seen[imp] = true
			imports = append(imports, imp)
		}
	}
	data := struct {
		Names
		Imports, Columns []string
	}{n, imports, columns}
	if err := g.write("app/models/"+n.File+".ts", "model.ts", data); err != nil {
		return err
	}
	return g.register("app/models/index.ts", "models", fmt.Sprintf("export * from %q;", "./"+n.File))
}

// Validator creates app/validators/<name>.ts with Zod rules from the fields.
func (g *Generator) Validator(name string, fields []Field) error {
	n := NamesFor(name)
	var rules []string
	for _, f := range fields {
		if f.Owner() && g.P.Has("auth") {
			continue // set from the signed-in user, never from input
		}
		rules = append(rules, f.RuleCode())
	}
	if len(rules) == 0 {
		rules = []string{"// title: z.string().min(1)"}
	}
	data := struct {
		Names
		Rules []string
	}{n, rules}
	return g.write("app/validators/"+n.File+".ts", "validator.ts", data)
}

type controllerData struct {
	Names
	Policies bool
	Owned    bool
}

func (g *Generator) ctrlData(name string) controllerData {
	n := NamesFor(name)
	owned := false
	if src, err := os.ReadFile(g.P.Path("app/models", n.File+".ts")); err == nil {
		owned = strings.Contains(string(src), "userId:") && g.P.Has("auth")
	}
	return controllerData{Names: n, Policies: g.P.Has("auth"), Owned: owned}
}

// Controller creates app/controllers/<name>.controller.ts.
func (g *Generator) Controller(name string, resource bool) error {
	name = Strip(Pascal(name), "Controller")
	d := g.ctrlData(name)
	stub := "controller.ts"
	if resource {
		stub = "controller.resource.ts"
		if !g.P.Exists("app/models/" + d.File + ".ts") {
			g.Notes = append(g.Notes, fmt.Sprintf("resource controller expects app/models/%s.ts — run `forge make:model %s`", d.File, d.Name))
		}
		if !g.P.Exists("app/validators/" + d.File + ".ts") {
			g.Notes = append(g.Notes, fmt.Sprintf("resource controller expects app/validators/%s.ts — run `forge make:request %s`", d.File, d.Name))
		}
	}
	return g.write("app/controllers/"+d.File+".controller.ts", stub, d)
}

// Routes registers a resource controller in routes/api.ts.
func (g *Generator) Routes(name string) error {
	n := NamesFor(Strip(Pascal(name), "Controller"))
	if err := g.register("routes/api.ts", "imports",
		fmt.Sprintf("import %sController from %q;", n.Name, "~/app/controllers/"+n.File+".controller")); err != nil {
		return err
	}
	return g.register("routes/api.ts", "routes", fmt.Sprintf("r.resource(%q, %sController);", "/"+n.Plural, n.Name))
}

// Policy creates app/policies/<name>.ts and registers it.
func (g *Generator) Policy(name string) error {
	if !g.P.Has("auth") {
		return fmt.Errorf("policies need the auth feature — run `forge install:auth`")
	}
	name = Strip(Pascal(name), "Policy")
	d := g.ctrlData(name)
	if err := g.write("app/policies/"+d.File+".ts", "policy.ts", d); err != nil {
		return err
	}
	if err := g.register("app/policies/index.ts", "imports", fmt.Sprintf("import %sPolicy from %q;", d.Name, "./"+d.File)); err != nil {
		return err
	}
	return g.register("app/policies/index.ts", "policies", d.Name+"Policy,")
}

// Middleware creates app/middleware/<name>.ts, usable as the "<name>" alias.
func (g *Generator) Middleware(name string) error {
	n := NamesFor(name)
	data := struct {
		Names
		Alias string
	}{n, n.Kebab}
	if err := g.write("app/middleware/"+n.File+".ts", "middleware.ts", data); err != nil {
		return err
	}
	if err := g.register("app/middleware/index.ts", "imports", fmt.Sprintf("import %s from %q;", n.Camel, "./"+n.File)); err != nil {
		return err
	}
	return g.register("app/middleware/index.ts", "middleware", fmt.Sprintf("%q: %s,", n.Kebab, n.Camel))
}

// Job creates app/jobs/<name>.ts and registers it with the worker.
func (g *Generator) Job(name string) error {
	n := NamesFor(name)
	if err := g.write("app/jobs/"+n.File+".ts", "job.ts", n); err != nil {
		return err
	}
	if err := g.register("app/jobs/index.ts", "imports", fmt.Sprintf("import %s from %q;", n.Name, "./"+n.File)); err != nil {
		return err
	}
	return g.register("app/jobs/index.ts", "jobs", n.Name+",")
}

// Seeder creates database/seeders/<name>.ts and calls it from the main seeder.
func (g *Generator) Seeder(name string) error {
	n := NamesFor(Strip(Pascal(name), "Seeder"))
	if err := g.write("database/seeders/"+n.File+".ts", "seeder.ts", n); err != nil {
		return err
	}
	if err := g.register("database/seeders/index.ts", "imports", fmt.Sprintf("import %sSeeder from %q;", n.Name, "./"+n.File)); err != nil {
		return err
	}
	return g.register("database/seeders/index.ts", "seeders", fmt.Sprintf("await %sSeeder();", n.Name))
}

// Test creates tests/<name>.test.ts.
func (g *Generator) Test(name string, resource bool) error {
	n := NamesFor(Strip(Pascal(name), "Test"))
	stub := "test.ts"
	if resource {
		stub = "test.resource.ts"
	}
	return g.write("tests/"+n.File+".test.ts", stub, g.ctrlData(n.Name))
}

// Tool creates an MCP tool in app/mcp/tools. With resource, it creates CRUD
// tools for the model of the same name instead.
func (g *Generator) Tool(name string, resource bool) error {
	if !g.P.Has("mcp") {
		return fmt.Errorf("tools need the mcp feature — run `forge install:mcp`")
	}
	name = Strip(Pascal(name), "Tool")
	d := g.ctrlData(name)
	data := struct {
		controllerData
		Tool, TableSingular, TablePlural, PluralWords string
	}{d, Snake(name), Snake(name), d.Table, strings.ReplaceAll(d.Table, "_", " ")}
	stub := "tool.ts"
	register := d.Camel + ","
	if resource {
		stub = "tool.resource.ts"
		register = "..." + d.Camel + "Tools,"
		for _, dep := range []string{"app/models/" + d.File + ".ts", "app/validators/" + d.File + ".ts", "app/policies/" + d.File + ".ts"} {
			if !g.P.Exists(dep) {
				g.Notes = append(g.Notes, fmt.Sprintf("%s tools expect %s — run `forge make:resource %s ...`", d.Name, dep, d.Name))
			}
		}
	}
	if err := g.write("app/mcp/tools/"+d.File+".ts", stub, data); err != nil {
		return err
	}
	importName := d.Camel
	if resource {
		importName = d.Camel + "Tools"
	}
	if err := g.register("app/mcp/index.ts", "imports", fmt.Sprintf("import %s from %q;", importName, "./tools/"+d.File)); err != nil {
		return err
	}
	return g.register("app/mcp/index.ts", "tools", register)
}

// Resource scaffolds model, validator, controller, policy, routes and a test.
func (g *Generator) Resource(name string, fields []Field) error {
	steps := []func() error{
		func() error { return g.Model(name, fields) },
		func() error { return g.Validator(name, fields) },
		func() error { return g.Controller(name, true) },
		func() error {
			if g.P.Has("auth") {
				return g.Policy(name)
			}
			return nil
		},
		func() error { return g.Routes(name) },
		func() error { return g.Test(name, true) },
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// PublishStubs copies the built-in stubs into <project>/stubs for customisation.
func (g *Generator) PublishStubs() error {
	for _, name := range scaffold.StubNames() {
		src, err := scaffold.StubSource("", name)
		if err != nil {
			return err
		}
		rel := filepath.Join("stubs", name+".tmpl")
		if err := project.WriteFile(g.P.Path(rel), src, g.Force); err != nil {
			if g.Force {
				return err
			}
			continue
		}
		g.Created = append(g.Created, rel)
	}
	return nil
}
