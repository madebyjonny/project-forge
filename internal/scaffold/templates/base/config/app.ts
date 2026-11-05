import { z } from "zod";

// Validated once at boot: a missing or malformed variable stops the app from starting.
const Env = z.object({
  APP_NAME: z.string().default("forge-app"),
  NODE_ENV: z.enum(["development", "test", "production"]).default("development"),
  PORT: z.coerce.number().default(3000),
  APP_URL: z.url().default("http://localhost:3000"),
  APP_KEY: z.string().min(32, "APP_KEY must be at least 32 chars — run `forge key:generate`"),
  CORS_ORIGINS: z.string().default(""),
});

const parsed = Env.parse(process.env);

export default {
  name: parsed.APP_NAME,
  env: parsed.NODE_ENV,
  port: parsed.PORT,
  url: parsed.APP_URL,
  key: parsed.APP_KEY,
  debug: parsed.NODE_ENV !== "production",
  cors: {
    origins: [parsed.APP_URL, ...parsed.CORS_ORIGINS.split(",").map((o) => o.trim()).filter(Boolean)],
  },
};
