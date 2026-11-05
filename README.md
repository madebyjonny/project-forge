# ⚒ forge

An artisan-style CLI for building Bun backends the Laravel way — one global
binary that scaffolds and manages a **Bun · Hono · Drizzle · Zod** stack, with
optional **Better Auth + CASL**, an **OAuth-secured MCP server** for AI clients,
**WebSockets**, **BullMQ queues** and **Docker**

Every command works two ways:

- **interactive** — a Bubble Tea UI (wizards, prompts, a command palette when you run `forge` with no arguments inside a project)
- **one-liners** — flags for everything; add `-n` / `--no-interaction` (or run in CI) to never prompt

## Install

```bash
make install            # builds and copies to /usr/local/bin/forge (PREFIX=... to change)
# or
make install-go         # go install → ~/go/bin/forge
```

Requires Go 1.23+ to build and [Bun](https://bun.sh) to run generated projects.

## Quick start

```bash
forge new                                   # interactive wizard
forge new blog                              # bare: hello-world routes on SQLite
forge new blog --preset auth                # + sessions, OAuth, policies
forge new blog --auth --realtime --db postgres
forge new blog --mcp                        # + an MCP server for VS Code / Codex / ChatGPT (adds auth)
forge new blog --preset full -n             # auth + mcp + realtime + queue + docker, no prompts

cd blog
forge serve                                 # http://localhost:3000 → {"message":"Hello from blog!"}
forge make:resource Post title:string body:text published:boolean:default=false user:references
forge route:list
forge test
```

## Commands

| Group | Command | |
| --- | --- | --- |
| Project | `new [name]` | `--preset bare\|auth\|full`, `--auth --mcp --realtime --queue --docker`, `--db sqlite\|postgres`, `--no-install`, `--no-git` |
| | `about` | Environment, drivers, stack and features |
| | `key:generate` | New `APP_KEY` (`--show`, `--force`) |
| Run | `serve` (`dev`) | Hot-reloading dev server (`--port`, `--env`, `--tunnel[=ngrok\|cloudflared]`, `--domain`) |
| | `start` | Production server |
| | `test [args]` | `bun test`, args pass through |
| | `typecheck` | `tsc --noEmit` |
| | `route:list` | Route table (`--path`, `--json`) |
| | `queue:work` | Start workers (`--concurrency`) |
| Generators | `make:model Name fields…` | `-m` migrate, `-c` controller+routes, `-r` request, `-a` everything |
| | `make:resource Name fields…` | Model + request + controller + policy + routes + test, migrated |
| | `make:controller Name` | `--resource`, `--routes` |
| | `make:request Name fields…` | Zod validator |
| | `make:policy Name` | CASL policy, auto-registered |
| | `make:middleware Name` | Named middleware, usable as `"name"` in routes |
| | `make:job Name` | Queued job, auto-registered with the worker |
| | `make:tool Name` | MCP tool (`--resource` for policy-checked CRUD tools); `make:resource … --mcp` does both |
| | `make:seeder Name`, `make:test Name` | |
| | `stub:publish` | Copy generator stubs to `./stubs` to customise |
| Database | `make:migration [name]` | Generate SQL from model changes, for **both** SQLite and Postgres |
| | `migrate` | `--fresh`, `--seed`, `--env production` |
| | `migrate:fresh` | Drop everything and re-run (`--seed`) |
| | `db:seed`, `db:studio` | |
| | `db:use sqlite\|postgres` | Switch driver for an environment (`--env`, `--url`) |
| MCP | `mcp:connect` | Configure VS Code (`--vscode`), Codex (`--codex`), ChatGPT (`--chatgpt`); `--url` sets the public URL |
| | `mcp:inspect` | Open the MCP Inspector |
| Features | `install [features…]`, `install:auth`, `install:mcp`, `install:realtime`, `install:queue`, `install:docker` | Add to an existing project (requirements like auth are added automatically) |

### Field syntax

`name:type[:modifier…]` — types `string text integer float boolean timestamp json references`,
modifiers `nullable` (or `type?`), `unique`, `default=value`.

```bash
forge make:model Comment body:text post:references user:references author:references:user published_at:datetime?
```

`x:references` becomes `xId` → `xs.id` with cascade delete. A `user:references`
field in an auth project makes the record *owned*: the controller sets `userId`
from the session and the policy only lets the owner update/delete.

## MCP server (VS Code, Codex, ChatGPT)

`forge install:mcp` (or `forge new app --mcp`) adds a [Model Context Protocol](https://modelcontextprotocol.io)
endpoint at `/mcp`, protected by OAuth 2.1 from your own Better Auth users — no API keys to paste.

```bash
forge new notes --mcp
cd notes
forge make:resource Note title:string body:text user:references --mcp   # REST + MCP tools, same policies
forge serve
forge mcp:connect --vscode            # writes .vscode/mcp.json → click Start, sign in, approve
```

Writing a tool is one function; it runs as the signed-in user:

```ts
// app/mcp/tools/summarize-notes.ts   (forge make:tool SummarizeNotes)
export default tool(
  "summarize_notes",
  { description: "Summarise my pinned notes.", input: { limit: z.number().default(5) }, readOnly: true },
  async ({ limit }, { user, authorize }) => {
    authorize("read", "Note");                       // same CASL policies as your controllers
    return Note.where({ userId: user.id, pinned: true });
  },
);
```

How a client connects:

1. It calls `/mcp`, gets a `401` pointing at `/.well-known/oauth-protected-resource`, and discovers the auth server.
2. It identifies itself with a Client ID Metadata Document (VS Code) or registers dynamically (ChatGPT and others).
3. The user signs in / signs up on `/oauth/sign-in` and approves on `/oauth/consent` (plain HTML in `core/mcp.ts` — restyle freely).
4. It receives an access token bound to `APP_URL/mcp` (PKCE, refresh tokens via `offline_access`) and calls tools.

| Client | How |
| --- | --- |
| VS Code (Copilot Chat, any model incl. GPT) | `forge mcp:connect --vscode`, open `.vscode/mcp.json`, click **Start** |
| Codex (ChatGPT's VS Code extension & CLI) | `forge mcp:connect --codex`, then `codex mcp login <name>` |
| ChatGPT (web/desktop connectors) | Needs public HTTPS: `forge serve --tunnel`, then add the printed `/mcp` URL as a connector with OAuth |

`forge serve --tunnel` starts ngrok or cloudflared and runs the app with `APP_URL` set to the public URL
(so the OAuth issuer and token audience match what remote clients see) without touching `.env`.
Use `--domain my-app.ngrok.app` for a stable URL so you don't have to re-add the connector.
Set `MCP_DYNAMIC_CLIENT_REGISTRATION=false` to only accept clients that publish a metadata document.

## Environments & databases

SQLite is the default. Each environment picks its driver from its own env file,
just like Laravel — no code changes:

```bash
forge db:use postgres --env production --url postgres://user:pass@host:5432/app
forge migrate --env production              # NODE_ENV=production → loads .env.production
```

Models are written once with dialect-neutral helpers (`t.string`, `t.boolean`,
`t.timestamp`, `t.foreignId`, …) and `make:migration` generates SQL for both
dialects, so switching is just an env change.

## What a generated project looks like

```
app/
  controllers/   home.controller.ts — plain objects of actions
  models/        tables + Post.find/where/create/update/delete/paginate
  validators/    Zod request schemas
  policies/      CASL rules                         (auth)
  middleware/    named middleware
  jobs/          queued jobs
  mcp/           MCP tools                          (mcp)
routes/          web.ts, api.ts (/api), channels.ts (realtime)
config/          typed config from .env
database/        migrations/{sqlite,postgres}, seeders
core/            the thin framework layer
tests/           bun test, in-memory SQLite, migrated per run
```

```ts
// routes/api.ts
export default routes((r) => {
  r.resource("/posts", PostController);
  r.get("/me", "auth", AuthController.me);
  r.group({ prefix: "/admin", middleware: ["auth", "role:admin"] }, (r) => {
    r.post("/posts/:id/publish", "can:update,Post", PostController.publish);
  });
});

// app/controllers/post.controller.ts
export default controller({
  async update(c) {
    const post = await Post.findOrFail(c.req.param("id")!);
    authorize(c, "update", post);                       // 403 unless the policy allows it
    const data = await validate(c, PostRequest.partial()); // 422 with { errors: { field: [...] } }
    return c.json(await Post.update(post.id, data));
  },
});
```

## Design: commands stay the same when the tools change

- **App code imports `core/`, never the libraries.** `user(c)`, `validate()`,
  `authorize()`, `dispatch()`, `broadcast()`, `Post.find()`, the `"auth"` middleware —
  these are forge's API. Better Auth lives only in `core/auth.ts`, CASL in
  `core/policy.ts`, BullMQ in `core/queue.ts`. Swap one by rewriting that file.
- **The CLI only calls `forge:*` scripts in `package.json`.** `forge migrate` runs
  `forge:migrate`, `forge serve` runs `forge:dev`, and so on. Replace Drizzle
  with something else and point the scripts at new tooling — the commands don't change.
- **`forge.json` records the stack** (`orm: drizzle`, `auth: better-auth`, …) and
  installed features. Generators read it, and project-level `stubs/*.tmpl`
  override the built-in templates.
- **Features are plugins.** `bootstrap/features.ts` is generated from `forge.json`;
  each feature registers routes, middleware aliases or server options through the
  same small `Feature` interface. Features that extend auth (like MCP) contribute
  Better Auth plugins and tables through `bootstrap/auth.ts`, so `core/auth.ts`
  never needs editing.

## Developing forge

```
main.go
internal/
  cli/        cobra commands
  ui/         lipgloss theme, Bubble Tea task runner + command palette
  scaffold/   embedded templates: base project, features, stubs
  gen/        make:* generators, naming, field parsing
  project/    forge.json, .env, package.json, registry editing
  runner/     runs the project's forge:* scripts
```

`make test` runs the Go tests. Generated projects are checked with `forge typecheck` and `forge test`.
