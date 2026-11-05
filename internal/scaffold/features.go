package scaffold

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/madebyjonny/forge/internal/project"
)

type EnvVar struct{ Key, Value, Example string }

type Insert struct{ File, Marker, Line string }

// Feature is an installable capability. App code talks to core/<feature>.ts, so
// the library behind a feature can change without changing commands or app code.
type Feature struct {
	Name        string
	Title       string
	Description string
	Stack       map[string]string // capability -> implementation recorded in forge.json
	Deps        map[string]string
	DevDeps     map[string]string
	EnvSection  string
	Env         []EnvVar
	Requires    []string // features installed first
	Boot        []string // "import x from ...|x()" entries for bootstrap/features.ts
	AuthBoot    []string // same format, for bootstrap/auth.ts (auth plugins)
	Models      []string // modules re-exported from app/models/index.ts
	Inserts     []Insert
	Files       bool // has templates/features/<name>
	Migrations  bool // adds tables → regenerate migrations
	Notes       []string
}

var Features = []Feature{
	{
		Name:        "auth",
		Title:       "Authentication + policies",
		Description: "Sessions, email/password & OAuth (Better Auth) with CASL policies",
		Stack:       map[string]string{"auth": "better-auth", "policies": "casl"},
		Deps:        deps("better-auth", "@casl/ability"),
		EnvSection:  "Auth",
		Env: []EnvVar{
			{Key: "AUTH_SECRET", Value: "{{secret}}", Example: ""},
			{Key: "GITHUB_CLIENT_ID", Value: "", Example: ""},
			{Key: "GITHUB_CLIENT_SECRET", Value: "", Example: ""},
		},
		Boot: []string{
			`import auth from "~/core/auth";|auth()`,
			`import policies from "~/core/policy";|policies()`,
		},
		Models: []string{"./user", "./auth"},
		Inserts: []Insert{
			{"routes/api.ts", "imports", `import AuthController from "~/app/controllers/auth.controller";`},
			{"routes/api.ts", "routes", `r.get("/me", "auth", AuthController.me);`},
		},
		Files:      true,
		Migrations: true,
		Notes: []string{
			"Auth endpoints: POST /api/auth/sign-up/email, /api/auth/sign-in/email, GET /api/me",
			"Guard routes with the \"auth\" middleware and policies with \"can:update,Post\"",
		},
	},
	{
		Name:        "realtime",
		Title:       "Realtime (WebSockets)",
		Description: "Authenticated sockets with channel authorization & broadcast()",
		Stack:       map[string]string{"realtime": "bun-ws"},
		EnvSection:  "Broadcasting",
		Env:         []EnvVar{{Key: "BROADCAST_CONNECTION", Value: "local", Example: "local"}},
		Boot:        []string{`import realtime from "~/core/realtime";|realtime()`},
		Files:       true,
		Notes: []string{
			"Connect to ws://localhost:3000/ws and send {\"type\":\"subscribe\",\"channel\":\"public.lobby\"}",
			"Authorize channels in routes/channels.ts; broadcast with broadcast(channel, event, data)",
		},
	},
	{
		Name:        "queue",
		Title:       "Queues (Redis)",
		Description: "Background jobs, retries & schedules via BullMQ",
		Stack:       map[string]string{"queue": "bullmq"},
		Deps:        deps("bullmq", "ioredis"),
		EnvSection:  "Queue",
		Env: []EnvVar{
			{Key: "QUEUE_CONNECTION", Value: "sync", Example: "redis"},
			{Key: "REDIS_URL", Value: "redis://localhost:6379", Example: "redis://localhost:6379"},
		},
		Notes: []string{
			"Set QUEUE_CONNECTION=redis, then run `forge queue:work`",
			"Create jobs with `forge make:job SendWelcomeEmail`",
		},
	},
	{
		Name:        "mcp",
		Title:       "MCP server",
		Description: "Model Context Protocol tools for AI clients, secured with OAuth",
		Requires:    []string{"auth"},
		Stack:       map[string]string{"mcp": "mcp-sdk", "oauth": "better-auth-mcp"},
		Deps:        deps("@better-auth/mcp", "@better-auth/cimd", "@better-auth/oauth-provider", "@modelcontextprotocol/server"),
		EnvSection:  "MCP",
		Env:         []EnvVar{{Key: "MCP_DYNAMIC_CLIENT_REGISTRATION", Value: "true", Example: "true"}},
		Boot:        []string{`import mcp from "~/core/mcp";|mcp()`},
		AuthBoot:    []string{`import { mcpAuth } from "~/core/mcp-auth";|mcpAuth()`},
		Models:      []string{"./oauth"},
		Files:       true,
		Migrations:  true,
		Notes: []string{
			"MCP endpoint: http://localhost:3000/mcp — connect a client with `forge mcp:connect`",
			"Add tools with `forge make:tool SendReport` or `forge make:tool Post --resource`",
			"ChatGPT needs a public HTTPS URL: `forge serve --tunnel`",
		},
	},
	{
		Name:        "docker",
		Title:       "Docker",
		Description: "Dockerfile + compose services for Postgres & Redis",
		Files:       true,
		Notes:       []string{"Start services with `docker compose up -d`"},
	},
}

func FeatureNames() []string {
	var out []string
	for _, f := range Features {
		out = append(out, f.Name)
	}
	return out
}

func GetFeature(name string) (Feature, error) {
	for _, f := range Features {
		if f.Name == name {
			return f, nil
		}
	}
	return Feature{}, fmt.Errorf("unknown feature %q (available: %s)", name, strings.Join(FeatureNames(), ", "))
}

// Resolve returns the features to install for names, with requirements first.
func Resolve(names []string) ([]Feature, error) {
	var out []Feature
	seen := map[string]bool{}
	var visit func(string) error
	visit = func(name string) error {
		if seen[name] {
			return nil
		}
		f, err := GetFeature(name)
		if err != nil {
			return err
		}
		seen[name] = true
		for _, r := range f.Requires {
			if err := visit(r); err != nil {
				return err
			}
		}
		out = append(out, f)
		return nil
	}
	for _, n := range names {
		if err := visit(n); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Install adds a feature's files, deps, env and registrations to a project.
// It's idempotent; package installation and migrations are left to the caller.
func Install(p *project.Project, f Feature) error {
	for _, r := range f.Requires {
		if !p.Has(r) {
			return fmt.Errorf("%s requires the %s feature — install it first", f.Name, r)
		}
	}
	if len(f.AuthBoot) > 0 {
		src, err := os.ReadFile(p.Path("core/auth.ts"))
		if err != nil {
			return err
		}
		if !strings.Contains(string(src), "bootstrap/auth") {
			return fmt.Errorf("core/auth.ts predates auth extensions — copy the AuthExtension wiring from a new project (`forge new tmp --preset auth --no-install`) before installing %s", f.Name)
		}
	}
	if f.Files {
		if _, err := copyTree("features/"+f.Name, p.Root, false); err != nil {
			return err
		}
	}

	pkg, err := project.OpenPackageJSON(p.Path("package.json"))
	if err != nil {
		return err
	}
	pkg.AddDeps(f.Deps, false)
	pkg.AddDeps(f.DevDeps, true)
	if err := pkg.Save(); err != nil {
		return err
	}

	if len(f.Env) > 0 {
		for _, file := range []string{".env", ".env.example"} {
			env, err := project.OpenEnv(p.Path(file))
			if err != nil {
				return err
			}
			env.Section(f.EnvSection)
			for _, v := range f.Env {
				value := v.Value
				if file == ".env.example" {
					value = v.Example
				} else if value == "{{secret}}" {
					value = Secret()
				}
				env.Default(v.Key, value)
			}
			if err := env.Save(); err != nil {
				return err
			}
		}
	}

	for _, m := range f.Models {
		if err := project.InsertBefore(p.Path("app/models/index.ts"), "models", fmt.Sprintf("export * from %q;", m)); err != nil {
			return err
		}
	}
	for _, in := range f.Inserts {
		if err := project.InsertBefore(p.Path(in.File), in.Marker, in.Line); err != nil {
			return err
		}
	}

	if !slices.Contains(p.Manifest.Features, f.Name) {
		p.Manifest.Features = append(p.Manifest.Features, f.Name)
	}
	if p.Manifest.Stack == nil {
		p.Manifest.Stack = map[string]string{}
	}
	for k, v := range f.Stack {
		p.Manifest.Stack[k] = v
	}
	if err := p.Save(); err != nil {
		return err
	}
	return WriteBootstrap(p)
}

// WriteBootstrap regenerates bootstrap/features.ts (and bootstrap/auth.ts) from forge.json.
func WriteBootstrap(p *project.Project) error {
	if err := writeRegistry(p, "bootstrap/features.ts", "Feature", "~/core/app", func(f Feature) []string { return f.Boot }); err != nil {
		return err
	}
	if !p.Has("auth") {
		return nil
	}
	return writeRegistry(p, "bootstrap/auth.ts", "AuthExtension", "~/core/auth", func(f Feature) []string { return f.AuthBoot })
}

func writeRegistry(p *project.Project, rel, typ, from string, entries func(Feature) []string) error {
	var imports, calls []string
	for _, f := range Features {
		if !p.Has(f.Name) {
			continue
		}
		for _, e := range entries(f) {
			imp, call, _ := strings.Cut(e, "|")
			imports = append(imports, imp)
			calls = append(calls, call)
		}
	}
	var b strings.Builder
	b.WriteString("// Generated by forge from forge.json — manage with `forge install:<feature>`.\n")
	fmt.Fprintf(&b, "import type { %s } from %q;\n", typ, from)
	for _, imp := range imports {
		b.WriteString(imp + "\n")
	}
	fmt.Fprintf(&b, "\nexport default [%s] satisfies %s[] as %s[];\n", strings.Join(calls, ", "), typ, typ)
	return project.WriteFile(p.Path(rel), []byte(b.String()), true)
}
