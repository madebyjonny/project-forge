import { env } from "../core/env";

const appUrl = env("APP_URL", "http://localhost:3000").replace(/\/+$/, "");
const path = "/mcp";

export default {
  name: env("MCP_NAME") ?? env("APP_NAME", "forge-app"),
  version: env("MCP_VERSION", "1.0.0"),
  path,
  // The public URL clients connect to. Tokens are bound to it, so it must match
  // what you paste into VS Code / ChatGPT (set APP_URL to your tunnel/domain).
  resource: `${appUrl}${path}`,
  // Pages the OAuth flow sends users to (served by core/mcp.ts — restyle freely).
  loginPage: "/oauth/sign-in",
  consentPage: "/oauth/consent",
  scopes: ["openid", "profile", "email", "offline_access"],
  // Where /mcp fetches the token signing keys. Defaults to this process over loopback
  // so verification never leaves the machine (tunnels/proxies don't matter).
  jwksUrl: () => env("MCP_JWKS_URL") ?? `http://127.0.0.1:${env("PORT", 3000)}/api/auth/jwks`,
  // Let MCP clients register themselves (needed by ChatGPT and older clients).
  // Clients that publish a Client ID Metadata Document (e.g. VS Code) work either way.
  dynamicClientRegistration: env("MCP_DYNAMIC_CLIENT_REGISTRATION", true),
};
