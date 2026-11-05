import { Hono } from "hono";
import { cors } from "hono/cors";
import { HTTPException } from "hono/http-exception";
import { logger } from "hono/logger";
import { secureHeaders } from "hono/secure-headers";
import config from "../config/app";
import { HttpError, ValidationError } from "./errors";
import type { AppEnv } from "./http";
import { alias, type Middleware } from "./middleware";
import { Router } from "./router";

export type App = Hono<AppEnv>;

/** A pluggable capability (auth, realtime, ...). Swap implementations without touching app code. */
export interface Feature {
  name: string;
  setup?: (app: App) => void;
  /** Extra Bun.serve options, e.g. websocket handlers. */
  serve?: Record<string, unknown>;
}

export interface AppOptions {
  features?: Feature[];
  middleware?: Record<string, Middleware>;
  routes: Record<string, { prefix: string; define: (r: Router) => void }>;
}

export function createApp({ features = [], middleware = {}, routes }: AppOptions): App {
  const app = new Hono<AppEnv>();

  if (config.env === "development") app.use("*", logger());
  app.use("*", secureHeaders());
  app.use("*", cors({ origin: config.cors.origins, credentials: true }));
  app.use("*", async (c, next) => {
    c.set("user", null);
    c.set("session", null);
    await next();
  });

  for (const [name, fn] of Object.entries(middleware)) alias(name, fn);
  for (const feature of features) feature.setup?.(app);
  for (const { prefix, define } of Object.values(routes)) define(new Router(app, prefix));

  app.onError((err, c) => {
    if (err instanceof ValidationError) return c.json({ message: err.message, errors: err.errors }, 422);
    if (err instanceof HttpError) return c.json({ message: err.message, ...err.details }, err.status);
    if (err instanceof HTTPException) return err.getResponse();
    console.error(err);
    return c.json(config.debug ? { message: err.message, stack: err.stack?.split("\n") } : { message: "Server Error" }, 500);
  });
  app.notFound((c) => c.json({ message: "Not found" }, 404));

  return app;
}

/** The Bun server definition: `export default serve(app)`. */
export function serve(app: App, features: Feature[] = []) {
  return Object.assign({ port: config.port, fetch: app.fetch }, ...features.map((f) => f.serve ?? {}));
}
