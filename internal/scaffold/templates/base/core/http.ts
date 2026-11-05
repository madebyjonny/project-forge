import type { Context, Next } from "hono";
import type { z } from "zod";
import { UnauthorizedError, ValidationError } from "./errors";

/** The authenticated user, independent of whichever auth library is installed. */
export interface AuthUser {
  id: string;
  name: string;
  email: string;
  role?: string | null;
  [key: string]: unknown;
}

export type AppEnv = {
  Variables: {
    user: AuthUser | null;
    session: unknown;
  };
};

export type Ctx = Context<AppEnv>;
export type Action = (c: Ctx) => Response | Promise<Response>;
export type { Next };

/** Group route actions into a controller. Just an object of functions. */
export function controller<T extends Record<string, Action>>(actions: T): T {
  return actions;
}

/** The current user, or null for guests. */
export function user(c: Ctx): AuthUser | null {
  return c.get("user") ?? null;
}

/** The current user, or a 401. */
export function userOrFail(c: Ctx): AuthUser {
  const u = user(c);
  if (!u) throw new UnauthorizedError();
  return u;
}

type Source = "body" | "query" | "params";

/**
 * Validate input against a Zod schema (≈ a Form Request). Throws a 422 with
 * Laravel-style `{ message, errors: { field: [..] } }` on failure.
 */
export async function validate<S extends z.ZodType>(c: Ctx, schema: S, source: Source = "body"): Promise<z.infer<S>> {
  const input = source === "query" ? c.req.query() : source === "params" ? c.req.param() : await readBody(c);
  const result = schema.safeParse(input);
  if (result.success) return result.data;

  const errors: Record<string, string[]> = {};
  for (const issue of result.error.issues) {
    const key = issue.path.join(".") || "_";
    (errors[key] ??= []).push(issue.message);
  }
  throw new ValidationError(errors);
}

async function readBody(c: Ctx): Promise<unknown> {
  const type = c.req.header("content-type") ?? "";
  if (type.includes("application/json")) return c.req.json().catch(() => ({}));
  if (type.includes("form")) return c.req.parseBody();
  return {};
}

/** Read pagination params from the query string. */
export function pageParams(c: Ctx) {
  return { page: Number(c.req.query("page") ?? 1), perPage: Number(c.req.query("per_page") ?? 15) };
}
