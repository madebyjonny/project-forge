import { afterAll, beforeAll, describe, expect, it } from "bun:test";
import { app } from "~/bootstrap/app";
import config from "~/config/mcp";

// Walks the whole flow an MCP client (VS Code, ChatGPT, …) performs.
const origin = new URL(config.resource).origin;
const json = { "content-type": "application/json", origin };
const redirectUri = "http://127.0.0.1:33418/callback";

const b64url = (bytes: ArrayBuffer | Uint8Array) => Buffer.from(bytes as ArrayBuffer).toString("base64url");
const signed = (url: string) => {
  const p = new URL(url, origin).searchParams;
  const keep = new Set(["sig", "ba_param", ...p.getAll("ba_param")]);
  return new URLSearchParams([...p].filter(([k]) => keep.has(k))).toString();
};
const rpc = (token: string | null, method: string, params: object = {}) =>
  app.request(config.path, {
    method: "POST",
    headers: {
      "content-type": "application/json",
      accept: "application/json, text/event-stream",
      "mcp-protocol-version": "2025-06-18",
      ...(token ? { authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ jsonrpc: "2.0", id: 1, method, params }),
  });

describe("mcp", () => {
  // /mcp verifies tokens against the app's signing keys over HTTP, so serve it.
  let server: ReturnType<typeof Bun.serve>;
  beforeAll(() => {
    server = Bun.serve({ port: 0, fetch: app.fetch });
    process.env.MCP_JWKS_URL = `http://127.0.0.1:${server.port}/api/auth/jwks`;
  });
  afterAll(() => server.stop(true));

  it("challenges unauthenticated clients with resource metadata", async () => {
    const res = await rpc(null, "tools/list");
    expect(res.status).toBe(401);
    expect(res.headers.get("www-authenticate")).toContain("/.well-known/oauth-protected-resource");

    for (const path of ["/.well-known/oauth-authorization-server", "/.well-known/openid-configuration"]) {
      expect(((await (await app.request(path)).json()) as any).token_endpoint).toContain("/oauth2/token");
    }
    const meta = (await (await app.request("/.well-known/oauth-protected-resource/mcp")).json()) as any;
    expect(meta.resource).toBe(config.resource);
  });

  it("authorizes a client end to end and calls a tool", async () => {
    // 1. dynamic client registration
    let res = await app.request("/api/auth/oauth2/register", {
      method: "POST",
      headers: json,
      body: JSON.stringify({ client_name: "Test Client", redirect_uris: [redirectUri], token_endpoint_auth_method: "none" }),
    });
    expect(res.status).toBe(201);
    const { client_id } = (await res.json()) as any;

    // 2. authorize with PKCE → login page
    const verifier = b64url(crypto.getRandomValues(new Uint8Array(32)));
    const challenge = b64url(await crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier)));
    const authorize = new URLSearchParams({
      response_type: "code", client_id, redirect_uri: redirectUri, state: "xyz",
      scope: "openid profile email offline_access", code_challenge: challenge, code_challenge_method: "S256",
      resource: config.resource,
    });
    res = await app.request(`/api/auth/oauth2/authorize?${authorize}`);
    const login = res.headers.get("location")!;
    expect(login).toContain(config.loginPage);
    expect((await app.request(login)).status).toBe(200);

    // 3. sign up on the login page → consent page
    res = await app.request("/api/auth/sign-up/email", {
      method: "POST",
      headers: json,
      body: JSON.stringify({ name: "Ada", email: "ada@mcp.test", password: "correct-horse-battery", oauth_query: signed(login) }),
    });
    expect(res.status).toBe(200);
    const cookie = res.headers.get("set-cookie")!;
    const consent = ((await res.json()) as any).url as string;
    expect(consent).toContain(config.consentPage);
    const consentHtml = await (await app.request(consent, { headers: { cookie } })).text();
    expect(consentHtml).toContain("Test Client");

    // 4. approve → redirect back with a code
    res = await app.request("/api/auth/oauth2/consent", {
      method: "POST",
      headers: { ...json, cookie },
      body: JSON.stringify({ accept: true, oauth_query: signed(consent) }),
    });
    const callback = new URL(((await res.json()) as any).url);
    expect(callback.searchParams.get("state")).toBe("xyz");

    // 5. exchange the code for an access token bound to the MCP resource
    res = await app.request("/api/auth/oauth2/token", {
      method: "POST",
      headers: { "content-type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "authorization_code", code: callback.searchParams.get("code")!, redirect_uri: redirectUri,
        client_id, code_verifier: verifier, resource: config.resource,
      }),
    });
    const token = (await res.json()) as any;
    expect(token.access_token).toBeString();

    // 6. use the MCP server as that user
    res = await rpc(token.access_token, "tools/call", { name: "whoami", arguments: {} });
    const body = await res.text();
    expect(res.status).toBe(200);
    expect(body).toContain("ada@mcp.test");
  });
});
