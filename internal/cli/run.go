package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/madebyjonny/forge/internal/project"
	"github.com/madebyjonny/forge/internal/runner"
	"github.com/madebyjonny/forge/internal/ui"
	"github.com/spf13/cobra"
)

func runCommands() []*cobra.Command {
	var serveEnv, tunnel, domain string
	var port int
	serve := &cobra.Command{
		Use:     "serve",
		Aliases: []string{"dev"},
		Short:   "Start the dev server with hot reload",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			vars := map[string]string{}
			if port > 0 {
				vars["PORT"] = strconv.Itoa(port)
			}
			fmt.Println(" " + ui.Logo())
			fmt.Println()
			if cmd.Flags().Changed("tunnel") || domain != "" {
				if port == 0 {
					port = envPort(p)
				}
				t, err := runner.StartTunnel(tunnel, port, domain)
				if err != nil {
					return err
				}
				defer t.Stop() // runs once the dev server exits (signals are forwarded to it)
				// The public URL becomes the app's URL for this run, so OAuth issuer,
				// redirects and the MCP resource all match what remote clients see.
				vars["APP_URL"] = t.URL
				ui.Success("Public URL via %s: %s", t.Provider, ui.Accent.Render(t.URL))
				if p.Has("mcp") {
					ui.Info("MCP endpoint for ChatGPT & co: %s", ui.Accent.Render(t.URL+mcpPath))
				}
				fmt.Println()
			}
			return script(p, "forge:dev", runner.Options{Env: serveEnv, Vars: vars})
		},
	}
	envFlag(serve, &serveEnv)
	serve.Flags().IntVar(&port, "port", 0, "port (default from .env)")
	serve.Flags().StringVar(&tunnel, "tunnel", "", "expose on a public https URL via ngrok or cloudflared")
	serve.Flags().Lookup("tunnel").NoOptDefVal = "auto"
	serve.Flags().StringVar(&domain, "domain", "", "reserved tunnel domain, e.g. my-app.ngrok.app (stable URL)")

	start := &cobra.Command{
		Use:   "start",
		Short: "Start the server without hot reload (production)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			env, _ := cmd.Flags().GetString("env")
			return script(p, "forge:start", runner.Options{Env: env})
		},
	}
	start.Flags().StringP("env", "e", "production", "environment")

	test := &cobra.Command{
		Use:                "test [args...]",
		Short:              "Run the test suite (args pass through to bun test)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			return script(p, "forge:test", runner.Options{Extra: args})
		},
	}

	typecheck := &cobra.Command{
		Use:   "typecheck",
		Short: "Type-check the project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			if err := script(p, "forge:typecheck", runner.Options{}); err != nil {
				return err
			}
			ui.Success("No type errors")
			return nil
		},
	}

	var asJSON bool
	var filter string
	routes := &cobra.Command{
		Use:   "route:list",
		Short: "List all registered routes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			out, err := runner.Capture(p, "forge:routes", runner.Options{})
			if err != nil {
				return err
			}
			if asJSON {
				fmt.Print(string(out))
				return nil
			}
			return printRoutes(out, filter)
		},
	}
	routes.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	routes.Flags().StringVar(&filter, "path", "", "only routes containing this path")

	work := &cobra.Command{
		Use:   "queue:work",
		Short: "Start a queue worker",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := mustProject()
			if err != nil {
				return err
			}
			env, _ := cmd.Flags().GetString("env")
			c, _ := cmd.Flags().GetInt("concurrency")
			return script(p, "forge:queue:work", runner.Options{Env: env, Extra: []string{"--concurrency", strconv.Itoa(c)}})
		},
	}
	work.Flags().StringP("env", "e", "", "environment")
	work.Flags().Int("concurrency", 5, "jobs processed in parallel")

	return []*cobra.Command{serve, start, test, typecheck, routes, work}
}

func envPort(p *project.Project) int {
	env, _ := project.OpenEnv(p.Path(".env"))
	if v, ok := env.Get("PORT"); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return 3000
}

type route struct {
	Method     string   `json:"method"`
	Path       string   `json:"path"`
	Middleware []string `json:"middleware"`
	Action     string   `json:"action"`
}

var methodColors = map[string]lipgloss.AdaptiveColor{
	"GET": ui.Blue, "POST": ui.Green, "PUT": ui.Yellow, "PATCH": ui.Yellow, "DELETE": ui.Red,
}

func printRoutes(raw []byte, filter string) error {
	// The script may log before printing JSON; take the last line.
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	var routes []route
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &routes); err != nil {
		return fmt.Errorf("could not read routes: %w", err)
	}
	var rows [][]string
	for _, r := range routes {
		if filter != "" && !strings.Contains(r.Path, filter) {
			continue
		}
		method := lipgloss.NewStyle().Foreground(methodColors[r.Method]).Bold(true).Render(r.Method)
		rows = append(rows, []string{method, r.Path, r.Action, ui.Faint.Render(strings.Join(r.Middleware, ", "))})
	}
	fmt.Println(ui.Table([]string{"Method", "URI", "Action", "Middleware"}, rows))
	fmt.Println(ui.Faint.Render(fmt.Sprintf("  %d routes", len(rows))))
	return nil
}

func init() { add("run", runCommands()...) }
