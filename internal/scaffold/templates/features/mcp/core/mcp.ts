// MCP server (≈ an API for AI clients). Tools live in app/mcp and receive the
// signed-in user, so they can use the same policies as your controllers.
//
//   VS Code / Codex / ChatGPT ──OAuth──▶ /api/auth/oauth2/*  (sign-in + consent pages below)
//                             ──Bearer─▶ /mcp                (tools from app/mcp)
import { requireMcpAuth } from "@better-auth/mcp";
import { verifyOAuthQueryParams } from "@better-auth/oauth-provider";
import { createMcpHandler, McpServer } from "@modelcontextprotocol/server";
import { eq } from "drizzle-orm";
import { z } from "zod";
import tools from "../app/mcp";
import { users } from "../app/models/user";
import authConfig from "../config/auth";
import config from "../config/mcp";
import type { App, Feature } from "./app";
import { auth } from "./auth";
import { db } from "./db";
import { HttpError, ValidationError } from "./errors";
import { user as currentUser, type AuthUser, type Ctx } from "./http";
import { abilityFor } from "./policy";

// ---- tools -----------------------------------------------------------------

export interface ToolContext {
  user: AuthUser;
  /** Scopes granted to the connected client. */
  scopes: string[];
  clientId: string;
  can(action: string, subject: object | string): boolean;
  /** Throws a 403 (reported to the client as a tool error) unless allowed. */
  authorize(action: string, subject: object | string): void;
}

export interface ToolOptions<S extends z.ZodRawShape> {
  title?: string;
  description: string;
  input?: S;
  /** Hints for clients: read-only tools may run without asking the user first. */
  readOnly?: boolean;
  destructive?: boolean;
}

export interface Tool<S extends z.ZodRawShape = any> extends ToolOptions<S> {
  name: string;
  handle: (input: z.infer<z.ZodObject<S>>, ctx: ToolContext) => unknown;
}

/** Define an MCP tool. Return a string, any JSON value, or a raw MCP result. */
export function tool<S extends z.ZodRawShape = {}>(
  name: string,
  options: ToolOptions<S>,
  handle: (input: z.infer<z.ZodObject<S>>, ctx: ToolContext) => unknown,
): Tool<S> {
  return { name, ...options, handle };
}

async function runTool(t: Tool, input: unknown, ctx: ToolContext) {
  try {
    const out = await t.handle(input as never, ctx);
    if (out && typeof out === "object" && Array.isArray((out as { content?: unknown }).content)) return out as never;
    const text = typeof out === "string" ? out : JSON.stringify(out ?? null, null, 2);
    return { content: [{ type: "text" as const, text }] };
  } catch (err) {
    if (err instanceof ValidationError) {
      return { isError: true, content: [{ type: "text" as const, text: `${err.message}\n${JSON.stringify(err.errors)}` }] };
    }
    if (err instanceof HttpError) return { isError: true, content: [{ type: "text" as const, text: err.message }] };
    throw err;
  }
}

const mcpHandler = createMcpHandler(({ authInfo }) => {
  const ctx = authInfo?.extra?.ctx as ToolContext;
  const server = new McpServer({ name: config.name, version: config.version });
  for (const t of tools) {
    server.registerTool(
      t.name,
      {
        title: t.title,
        description: t.description,
        inputSchema: z.object(t.input ?? {}),
        annotations: { readOnlyHint: t.readOnly ?? false, destructiveHint: t.destructive ?? !t.readOnly },
      },
      (input: unknown) => runTool(t, input, ctx),
    );
  }
  return server;
});

async function loadUser(id: string): Promise<AuthUser | null> {
  const [row] = await (db as any).select().from(users).where(eq(users.id, id)).limit(1);
  return (row as AuthUser | undefined) ?? null;
}

function toolContext(u: AuthUser, scopes: string[], clientId: string): ToolContext {
  const ability = abilityFor(u);
  return {
    user: u,
    scopes,
    clientId,
    can: (action, subject) => ability.can(action, subject as never),
    authorize(action, subject) {
      if (!ability.can(action, subject as never)) throw new HttpError(403, "This action is unauthorized.");
    },
  };
}

// ---- OAuth screens ----------------------------------------------------------

const esc = (s: unknown) =>
  String(s ?? "").replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch]!);

/** The signed part of an OAuth redirect query (what Better Auth verifies). */
function signedQuery(url: string): string {
  const params = new URL(url).searchParams;
  const keep = new Set(["sig", "ba_param", ...params.getAll("ba_param")]);
  return new URLSearchParams([...params].filter(([key]) => keep.has(key))).toString();
}

const scopeLabels: Record<string, string> = {
  openid: "Confirm your identity",
  profile: "See your name and picture",
  email: "See your email address",
  offline_access: "Stay connected when you're not using it",
};

function page(title: string, body: string, script = "") {
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>${esc(title)} · ${esc(config.name)}</title>
<style>
  :root { color-scheme: light dark; --accent: #ff7a1a; --muted: #8a8f98; --line: color-mix(in srgb, currentColor 14%, transparent); }
  * { box-sizing: border-box; } body { margin: 0; min-height: 100vh; display: grid; place-items: center; font: 15px/1.5 system-ui, sans-serif; background: Canvas; }
  main { width: min(380px, calc(100% - 2rem)); padding: 2rem; border: 1px solid var(--line); border-radius: 14px; }
  h1 { font-size: 1.25rem; margin: 0 0 .25rem; } p { margin: .25rem 0 1rem; color: var(--muted); }
  label { display: block; font-size: .85rem; margin: .75rem 0 .25rem; }
  input { width: 100%; padding: .6rem .7rem; border: 1px solid var(--line); border-radius: 8px; font: inherit; background: transparent; color: inherit; }
  button { width: 100%; margin-top: 1rem; padding: .65rem; border: 0; border-radius: 8px; font: inherit; font-weight: 600; cursor: pointer; background: var(--accent); color: #1a1a1a; }
  button.secondary { background: transparent; color: inherit; border: 1px solid var(--line); }
  ul { padding-left: 1.1rem; margin: .5rem 0 0; } .err { color: #e5484d; min-height: 1.2em; margin: .75rem 0 0; }
  a { color: var(--accent); cursor: pointer; } .host { font-family: ui-monospace, monospace; font-size: .85rem; }
</style></head><body><main>${body}</main>${script ? `<script>${script}</script>` : ""}</body></html>`;
}

function signInPage(c: Ctx) {
  const oauthQuery = signedQuery(c.req.url);
  const base = authConfig.basePath;
  return c.html(
    page(
      "Sign in",
      `<h1 id="title">Sign in</h1><p>to continue to <b>${esc(config.name)}</b></p>
      <form id="form">
        <div id="name-row" hidden><label for="name">Name</label><input id="name" autocomplete="name"></div>
        <label for="email">Email</label><input id="email" type="email" autocomplete="email" required>
        <label for="password">Password</label><input id="password" type="password" autocomplete="current-password" required minlength="8">
        <button id="submit">Sign in</button>
        <p class="err" id="err" role="alert"></p>
      </form>
      <p id="toggle-row">No account? <a id="toggle">Create one</a></p>`,
      `const q = ${JSON.stringify(oauthQuery)}; let signUp = false;
      const $ = (id) => document.getElementById(id);
      $("toggle").onclick = () => { signUp = !signUp; $("name-row").hidden = !signUp; $("title").textContent = $("submit").textContent = signUp ? "Create account" : "Sign in";
        $("toggle").textContent = signUp ? "Sign in instead" : "Create one"; $("password").autocomplete = signUp ? "new-password" : "current-password"; };
      $("form").onsubmit = async (e) => {
        e.preventDefault(); $("err").textContent = ""; $("submit").disabled = true;
        const body = { email: $("email").value, password: $("password").value, oauth_query: q };
        if (signUp) body.name = $("name").value || $("email").value.split("@")[0];
        const res = await fetch(${JSON.stringify(base)} + (signUp ? "/sign-up/email" : "/sign-in/email"), {
          method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify(body) });
        const data = await res.json().catch(() => ({}));
        $("submit").disabled = false;
        if (!res.ok) return ($("err").textContent = data.message || "Something went wrong");
        if (data.url) location.href = data.url; else $("err").textContent = "Signed in — return to your app.";
      };`,
    ),
  );
}

async function consentPage(c: Ctx) {
  const query = signedQuery(c.req.url);
  const { secret } = await auth.$context;
  if (!query || !(await verifyOAuthQueryParams(query, secret))) {
    return c.html(page("Link expired", "<h1>Link expired</h1><p>Start connecting again from your app.</p>"), 400);
  }
  const params = new URLSearchParams(query);
  if (!currentUser(c)) {
    return c.redirect(`${config.loginPage}?${query}`);
  }

  let client: { client_name?: string; name?: string } = {};
  try {
    client = await (auth.api as any).getOAuthClientPublic({ query: { client_id: params.get("client_id") }, headers: c.req.raw.headers });
  } catch {
    return c.html(page("Unknown app", "<h1>Unknown app</h1><p>This app isn't registered or has been disabled.</p>"), 404);
  }
  let host = "";
  try {
    host = new URL(params.get("redirect_uri") ?? "").host;
  } catch {}

  const scopes = (params.get("scope") ?? "").split(" ").filter(Boolean);
  const u = currentUser(c)!;
  return c.html(
    page(
      "Authorize",
      `<h1>${esc(client.client_name ?? client.name ?? "An app")} wants access</h1>
      <p>Signed in as <b>${esc(u.email)}</b>${host ? ` · returns to <span class="host">${esc(host)}</span>` : ""}</p>
      <p style="margin-bottom:0">It will be able to:</p>
      <ul>${scopes.map((s) => `<li>${esc(scopeLabels[s] ?? s)}</li>`).join("")}<li>Use the ${esc(config.name)} tools on your behalf</li></ul>
      <button id="allow">Allow</button><button id="deny" class="secondary">Deny</button>
      <p class="err" id="err" role="alert"></p>`,
      `const q = ${JSON.stringify(query)};
      async function decide(accept) {
        const res = await fetch(${JSON.stringify(authConfig.basePath + "/oauth2/consent")}, {
          method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ accept, oauth_query: q }) });
        const data = await res.json().catch(() => ({}));
        if (data.url) location.href = data.url; else document.getElementById("err").textContent = data.message || "Something went wrong";
      }
      document.getElementById("allow").onclick = () => decide(true);
      document.getElementById("deny").onclick = () => decide(false);`,
    ),
  );
}

// ---- feature ----------------------------------------------------------------

export default function mcpFeature(): Feature {
  return {
    name: "mcp",
    setup(app: App) {
      // OAuth discovery documents live at the domain root. Older clients probe the bare
      // paths, so alias them to the issuer's (path-inserted) documents.
      const issuerPath = authConfig.basePath;
      app.get("/.well-known/oauth-authorization-server", (c) =>
        auth.handler(new Request(new URL(`/.well-known/oauth-authorization-server${issuerPath}`, c.req.url).href)),
      );
      app.get("/.well-known/openid-configuration", (c) =>
        auth.handler(new Request(new URL(`${issuerPath}/.well-known/openid-configuration`, c.req.url).href)),
      );
      app.get("/.well-known/*", (c) => auth.handler(c.req.raw));

      app.get(config.loginPage, (c) => signInPage(c));
      app.get(config.consentPage, (c) => consentPage(c));

      const endpoint = requireMcpAuth(
        auth,
        async (request, claims) => {
          const u = await loadUser(String(claims.sub ?? ""));
          if (!u) return Response.json({ error: "invalid_token" }, { status: 401 });
          const scopes = String(claims.scope ?? "").split(" ").filter(Boolean);
          const clientId = String(claims.azp ?? claims.client_id ?? "");
          const token = request.headers.get("authorization")?.replace(/^\S+\s+/, "") ?? "";
          return mcpHandler.fetch(request, {
            authInfo: { token, clientId, scopes, extra: { ctx: toolContext(u, scopes, clientId) } },
          });
        },
        {
          resource: config.resource,
          get jwksUrl() {
            return config.jwksUrl();
          },
        },
      );
      app.all(config.path, (c) => endpoint(c.req.raw));
    },
  };
}
