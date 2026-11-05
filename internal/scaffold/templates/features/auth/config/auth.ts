import { env } from "../core/env";

const social = (id?: string, secret?: string) => (id && secret ? { clientId: id, clientSecret: secret } : undefined);

export default {
  secret: env("AUTH_SECRET") ?? env("APP_KEY", ""),
  url: env("APP_URL", "http://localhost:3000"),
  // Mounted under /api/auth: sign-up/email, sign-in/email, sign-out, get-session, ...
  basePath: "/api/auth",
  emailAndPassword: true,
  // Headers to read the client IP from (rate limiting, session metadata). Only list
  // headers your load balancer sets — clients can spoof them otherwise (≈ TrustProxies).
  ipHeaders: env("TRUSTED_IP_HEADERS", "x-forwarded-for,x-real-ip").split(",").map((h) => h.trim()),
  // Providers switch on when their credentials are present in .env.
  social: Object.fromEntries(
    Object.entries({
      github: social(env("GITHUB_CLIENT_ID"), env("GITHUB_CLIENT_SECRET")),
      google: social(env("GOOGLE_CLIENT_ID"), env("GOOGLE_CLIENT_SECRET")),
    }).filter(([, v]) => v),
  ) as Record<string, { clientId: string; clientSecret: string }>,
};
