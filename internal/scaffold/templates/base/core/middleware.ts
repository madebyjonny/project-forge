import type { MiddlewareHandler } from "hono";
import { HttpError, UnauthorizedError } from "./errors";
import { user, type AppEnv, type Ctx, type Next } from "./http";

/**
 * A middleware receives any `:arg1,arg2` from its alias, e.g. "throttle:60,1".
 * Return nothing to continue, or a Response to short-circuit.
 */
export type Middleware = (c: Ctx, next: Next, args: string[]) => Promise<Response | void> | Response | void;

export function defineMiddleware(fn: Middleware): Middleware {
  return fn;
}

export type MiddlewareRef = string | MiddlewareHandler<AppEnv>;

const aliases = new Map<string, Middleware>();

/** Register a named middleware usable as a string in routes. */
export function alias(name: string, fn: Middleware) {
  aliases.set(name, fn);
}

export function resolveMiddleware(ref: MiddlewareRef): MiddlewareHandler<AppEnv> {
  if (typeof ref !== "string") return ref;
  const [name = "", rawArgs = ""] = ref.split(/:(.*)/s);
  const fn = aliases.get(name);
  if (!fn) throw new Error(`Unknown middleware "${name}". Register it in app/middleware/index.ts`);
  const args = rawArgs ? rawArgs.split(",").map((a) => a.trim()) : [];
  return async (c, next) => {
    let called = false;
    const res = await fn(c, async () => {
      called = true;
      await next();
    }, args);
    if (res instanceof Response) return res;
    if (!called) await next();
  };
}

// ---- built-ins -------------------------------------------------------------

alias("auth", (c) => {
  if (!user(c)) throw new UnauthorizedError();
});

alias("guest", (c) => {
  if (user(c)) throw new HttpError(403, "Already authenticated");
});

alias("role", (c, _next, roles) => {
  const u = user(c);
  if (!u) throw new UnauthorizedError();
  if (!roles.includes(String(u.role))) throw new HttpError(403, "Forbidden");
});

// "throttle:limit,seconds" — in-memory, per user or IP. Use a shared store when scaling out.
const hits = new Map<string, { count: number; reset: number }>();
alias("throttle", (c, _next, [limit = "60", seconds = "60"]) => {
  const id = user(c)?.id ?? c.req.header("x-forwarded-for") ?? c.req.header("x-real-ip") ?? "anon";
  const key = `${c.req.routePath}:${id}`;
  const now = Date.now();
  const entry = hits.get(key);
  if (!entry || entry.reset < now) {
    hits.set(key, { count: 1, reset: now + Number(seconds) * 1000 });
    return;
  }
  if (++entry.count > Number(limit)) {
    c.header("Retry-After", String(Math.ceil((entry.reset - now) / 1000)));
    throw new HttpError(429, "Too Many Requests");
  }
});
