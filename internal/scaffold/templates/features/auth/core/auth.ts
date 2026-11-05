// The auth adapter. App code never imports the auth library directly — it uses
// user(c), the "auth" middleware and `can`, so this file is the only thing to
// replace if you swap Better Auth for something else.
import { betterAuth, type BetterAuthPlugin } from "better-auth";
import { drizzleAdapter } from "better-auth/adapters/drizzle";
import config from "../config/auth";
import database from "../config/database";
import { accounts, sessions, verifications } from "../app/models/auth";
import { users } from "../app/models/user";
import extensions from "../bootstrap/auth";
import type { Feature } from "./app";
import { db } from "./db";
import type { AuthUser } from "./http";
import { isTesting } from "./env";

/** What an installed feature (e.g. mcp) adds to the auth layer. */
export interface AuthExtension {
  plugins?: BetterAuthPlugin[];
  schema?: Record<string, unknown>;
  disabledPaths?: string[];
}

const extra = <K extends keyof AuthExtension>(key: K) => extensions.flatMap((e) => (e[key] ?? []) as any[]);

export const auth = betterAuth({
  database: drizzleAdapter(db, {
    provider: database.connection === "postgres" ? "pg" : "sqlite",
    schema: Object.assign(
      { user: users, session: sessions, account: accounts, verification: verifications },
      ...extensions.map((e) => e.schema ?? {}),
    ),
  }),
  secret: config.secret,
  baseURL: config.url,
  basePath: config.basePath,
  emailAndPassword: { enabled: config.emailAndPassword },
  socialProviders: config.social,
  user: {
    additionalFields: { role: { type: "string", defaultValue: "member", input: false } },
  },
  rateLimit: { enabled: !isTesting(), window: 60, max: 30 },
  advanced: { ipAddress: { ipAddressHeaders: config.ipHeaders } },
  plugins: extra("plugins"),
  disabledPaths: extra("disabledPaths"),
});

export default function authFeature(): Feature {
  return {
    name: "auth",
    setup(app) {
      app.on(["GET", "POST"], `${config.basePath}/*`, (c) => auth.handler(c.req.raw));

      // Resolve the session once per request so user(c) works everywhere.
      app.use("*", async (c, next) => {
        const session = await auth.api.getSession({ headers: c.req.raw.headers });
        c.set("user", (session?.user as AuthUser | undefined) ?? null);
        c.set("session", session?.session ?? null);
        await next();
      });
    },
  };
}
