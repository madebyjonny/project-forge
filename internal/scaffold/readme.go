package scaffold

import (
	"fmt"
	"strings"

	"github.com/madebyjonny/forge/internal/project"
)

func writeReadme(p *project.Project) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nBuilt with [forge](https://github.com/madebyjonny/forge) — Bun · Hono · Drizzle · Zod", p.Manifest.Name)
	if p.Has("auth") {
		b.WriteString(" · Better Auth · CASL")
	}
	b.WriteString(".\n\n")
	b.WriteString("```bash\nforge serve            # dev server with hot reload\nforge test             # run the test suite\nforge route:list       # see every route\nforge make:resource Post title:string body:text published:boolean\nforge migrate          # apply migrations\n```\n\n")
	b.WriteString(`## Layout

| Path | What lives there |
| --- | --- |
| ` + "`routes/`" + ` | ` + "`web.ts`" + `, ` + "`api.ts`" + ` (prefixed ` + "`/api`" + `)` + channelsRow(p) + ` |
| ` + "`app/controllers/`" + ` | Controllers — plain objects of actions |
| ` + "`app/models/`" + ` | Tables + model helpers (` + "`Post.find()`, `Post.paginate()`" + `) |
| ` + "`app/validators/`" + ` | Zod request schemas |
| ` + "`app/policies/`" + ` | Authorization rules |
| ` + "`app/middleware/`" + ` | Named middleware |
| ` + "`app/jobs/`" + ` | Background jobs |` + mcpRow(p) + `
| ` + "`config/`" + ` | Typed config read from ` + "`.env`" + ` |
| ` + "`database/`" + ` | Migrations (per dialect) and seeders |
| ` + "`core/`" + ` | The thin framework layer — app code imports from here, never from the libraries directly |

## Databases

SQLite by default. Switch any environment to Postgres without code changes:

` + "```bash" + `
forge db:use postgres --url postgres://user:pass@host:5432/db --env production
forge migrate --env production
` + "```" + `

Migrations are generated for both dialects by ` + "`forge make:migration`" + `.
`)
	return project.WriteFile(p.Path("README.md"), []byte(b.String()), true)
}

func mcpRow(p *project.Project) string {
	if p.Has("mcp") {
		return "\n| `app/mcp/` | MCP tools at `/mcp` — `forge mcp:connect` to use them from VS Code / Codex / ChatGPT |"
	}
	return ""
}

func channelsRow(p *project.Project) string {
	if p.Has("realtime") {
		return ", `channels.ts`"
	}
	return ""
}
