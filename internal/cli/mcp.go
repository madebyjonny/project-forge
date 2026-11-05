package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/madebyjonny/forge/internal/project"
	"github.com/madebyjonny/forge/internal/runner"
	"github.com/madebyjonny/forge/internal/ui"
	"github.com/spf13/cobra"
)

const mcpPath = "/mcp"

func requireMCP() (*project.Project, error) {
	p, err := mustProject()
	if err != nil {
		return nil, err
	}
	if !p.Has("mcp") {
		return nil, fmt.Errorf("this project has no MCP server — run `forge install:mcp`")
	}
	return p, nil
}

// appURL reads APP_URL from .env (what the server advertises as its issuer/resource).
func appURL(p *project.Project) string {
	env, _ := project.OpenEnv(p.Path(".env"))
	if v, ok := env.Get("APP_URL"); ok && v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:3000"
}

func mcpConnectCmd() *cobra.Command {
	var url, name string
	var vscode, codex, chatgpt bool
	cmd := &cobra.Command{
		Use:   "mcp:connect",
		Short: "Connect VS Code, Codex or ChatGPT to this project's MCP server",
		Example: `  forge mcp:connect                     # pick clients interactively
  forge mcp:connect --vscode            # write .vscode/mcp.json (Copilot Chat, any model)
  forge mcp:connect --codex             # add to ~/.codex/config.toml (ChatGPT Codex extension/CLI)
  forge mcp:connect --chatgpt --url https://my-app.ngrok.app`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := requireMCP()
			if err != nil {
				return err
			}
			if name == "" {
				name = p.Manifest.Name
			}
			if url != "" {
				url = strings.TrimRight(url, "/")
				if !strings.HasPrefix(url, "https://") {
					return fmt.Errorf("--url must be the public https:// base URL of the app")
				}
				env, err := project.OpenEnv(p.Path(".env"))
				if err != nil {
					return err
				}
				env.Set("APP_URL", url)
				if err := env.Save(); err != nil {
					return err
				}
				ui.Success("APP_URL set to %s %s", url, ui.Faint.Render("(restart `forge serve` to apply)"))
			}

			if !vscode && !codex && !chatgpt {
				if ui.Interactive() {
					var picks []string
					err := huh.NewMultiSelect[string]().Title("Which clients?").Options(
						huh.NewOption("VS Code — Copilot Chat (GPT, Claude, …)", "vscode").Selected(true),
						huh.NewOption("Codex — ChatGPT's VS Code extension & CLI", "codex"),
						huh.NewOption("ChatGPT — web/desktop connector", "chatgpt"),
					).Value(&picks).Run()
					if err != nil {
						return err
					}
					for _, pick := range picks {
						vscode = vscode || pick == "vscode"
						codex = codex || pick == "codex"
						chatgpt = chatgpt || pick == "chatgpt"
					}
				} else {
					vscode = true
				}
			}

			endpoint := appURL(p) + mcpPath
			ui.Header("mcp")
			fmt.Printf("  %s %s\n\n", ui.Faint.Render("endpoint"), ui.Accent.Render(endpoint))
			if vscode {
				if err := writeVSCodeConfig(p, name, endpoint); err != nil {
					return err
				}
				ui.Success("VS Code: wrote .vscode/mcp.json")
				ui.Hint(
					"Run "+ui.Code.Render("forge serve")+", open .vscode/mcp.json and click "+ui.Bold.Render("Start")+" above \""+name+"\"",
					"VS Code opens the sign-in page; approve access, then use the tools from Chat (agent mode)",
				)
			}
			if codex {
				path, err := writeCodexConfig(name, endpoint)
				if err != nil {
					return err
				}
				ui.Success("Codex: added [mcp_servers.%s] to %s", name, path)
				ui.Hint(
					"Run "+ui.Code.Render("forge serve")+", then "+ui.Code.Render("codex mcp login "+name)+" to sign in",
					"The ChatGPT/Codex VS Code extension picks it up from the same config",
				)
			}
			if chatgpt {
				public := endpoint
				local := strings.Contains(endpoint, "localhost") || strings.Contains(endpoint, "127.0.0.1")
				if local {
					public = "https://<public-url>" + mcpPath
				}
				lines := []string{
					"In ChatGPT: Settings → Apps & Connectors → enable Developer mode → Create",
					"MCP server URL: " + ui.Accent.Render(public) + "   Authentication: OAuth",
					"Save, then sign in and approve on your app's consent page",
				}
				if local {
					lines = append([]string{ui.Warning.Render("ChatGPT can't reach localhost.") + " Run " +
						ui.Code.Render("forge serve --tunnel") + " (or deploy) and use the MCP URL it prints"}, lines...)
				}
				ui.Success("ChatGPT")
				ui.Hint(lines...)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&url, "url", "", "public https base URL of the app (sets APP_URL)")
	cmd.Flags().StringVar(&name, "name", "", "server name shown in clients (default: project name)")
	cmd.Flags().BoolVar(&vscode, "vscode", false, "write .vscode/mcp.json")
	cmd.Flags().BoolVar(&codex, "codex", false, "add the server to ~/.codex/config.toml")
	cmd.Flags().BoolVar(&chatgpt, "chatgpt", false, "show ChatGPT connector steps")
	return cmd
}

func writeVSCodeConfig(p *project.Project, name, endpoint string) error {
	path := p.Path(".vscode", "mcp.json")
	cfg := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf(".vscode/mcp.json isn't plain JSON (comments?) — add this server by hand: %q: {\"type\": \"http\", \"url\": %q}", name, endpoint)
		}
	}
	servers, _ := cfg["servers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers[name] = map[string]any{"type": "http", "url": endpoint}
	cfg["servers"] = servers
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return project.WriteJSON(path, cfg)
}

// writeCodexConfig adds or updates [mcp_servers.<name>] in ~/.codex/config.toml.
func writeCodexConfig(name, endpoint string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".codex")
	if v := os.Getenv("CODEX_HOME"); v != "" {
		dir = v
	}
	path := filepath.Join(dir, "config.toml")
	data, _ := os.ReadFile(path)
	content := string(data)

	header := fmt.Sprintf("[mcp_servers.%s]", tomlKey(name))
	section := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(header) + `\s*\n((?:[^\[\n].*\n?|\n)*)`)
	if loc := section.FindStringSubmatchIndex(content); loc != nil {
		body := content[loc[2]:loc[3]]
		urlLine := regexp.MustCompile(`(?m)^url\s*=.*$`)
		if urlLine.MatchString(body) {
			body = urlLine.ReplaceAllString(body, fmt.Sprintf("url = %q", endpoint))
		} else {
			body = fmt.Sprintf("url = %q\n", endpoint) + body
		}
		content = content[:loc[2]] + body + content[loc[3]:]
	} else {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		content += fmt.Sprintf("%s\nurl = %q\n", header, endpoint)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return strings.Replace(path, home, "~", 1), nil
}

func tomlKey(s string) string {
	if regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(s) {
		return s
	}
	return fmt.Sprintf("%q", s)
}

func mcpInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp:inspect",
		Short: "Open the MCP Inspector against this project",
		RunE: func(cmd *cobra.Command, _ []string) error {
			p, err := requireMCP()
			if err != nil {
				return err
			}
			endpoint := appURL(p) + mcpPath
			ui.Info("In the Inspector choose Streamable HTTP, URL %s, then Connect to sign in", ui.Accent.Render(endpoint))
			return runner.Exec(p, []string{"x", "@modelcontextprotocol/inspector"}, runner.Options{})
		},
	}
}

func init() {
	root.AddGroup(&cobra.Group{ID: "mcp", Title: "MCP:"})
	add("mcp", mcpConnectCmd(), mcpInspectCmd())
}
