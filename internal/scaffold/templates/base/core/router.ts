import type { Hono } from "hono";
import type { Action, AppEnv } from "./http";
import { resolveMiddleware, type MiddlewareRef } from "./middleware";

type Method = "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "OPTIONS";
type ResourceAction = "index" | "show" | "store" | "update" | "destroy";
type Handlers = [...MiddlewareRef[], Action];

export interface RouteInfo {
  method: string;
  path: string;
  middleware: string[];
  action: string;
}

/** Every registered route — powers `forge route:list`. */
export const routeTable: RouteInfo[] = [];

export interface GroupOptions {
  prefix?: string;
  middleware?: MiddlewareRef[];
}

const join = (...parts: string[]) => "/" + parts.join("/").split("/").filter(Boolean).join("/");

export class Router {
  constructor(
    private app: Hono<AppEnv>,
    private prefix = "",
    private stack: MiddlewareRef[] = [],
  ) {}

  get = (path: string, ...h: Handlers) => this.add(["GET"], path, h);
  post = (path: string, ...h: Handlers) => this.add(["POST"], path, h);
  put = (path: string, ...h: Handlers) => this.add(["PUT"], path, h);
  patch = (path: string, ...h: Handlers) => this.add(["PATCH"], path, h);
  delete = (path: string, ...h: Handlers) => this.add(["DELETE"], path, h);
  match = (methods: Method[], path: string, ...h: Handlers) => this.add(methods, path, h);

  /** RESTful routes for a controller: index, show, store, update, destroy. */
  resource(
    path: string,
    ctrl: Partial<Record<ResourceAction, Action>>,
    opts: { only?: ResourceAction[]; except?: ResourceAction[]; middleware?: MiddlewareRef[] } = {},
  ) {
    const routes: [Method[], string, ResourceAction][] = [
      [["GET"], "", "index"],
      [["GET"], ":id", "show"],
      [["POST"], "", "store"],
      [["PUT", "PATCH"], ":id", "update"],
      [["DELETE"], ":id", "destroy"],
    ];
    for (const [methods, sub, name] of routes) {
      const action = ctrl[name];
      if (!action || opts.only?.includes(name) === false || opts.except?.includes(name)) continue;
      this.add(methods, join(path, sub), [...(opts.middleware ?? []), action], name);
    }
    return this;
  }

  /** Share a prefix and/or middleware across routes. */
  group(opts: GroupOptions | string, define: (r: Router) => void) {
    const o = typeof opts === "string" ? { prefix: opts } : opts;
    define(new Router(this.app, join(this.prefix, o.prefix ?? ""), [...this.stack, ...(o.middleware ?? [])]));
    return this;
  }

  /** A router whose routes all run the given middleware. */
  middleware(...refs: MiddlewareRef[]) {
    return new Router(this.app, this.prefix, [...this.stack, ...refs]);
  }

  private add(methods: Method[], path: string, handlers: Handlers, name?: string) {
    const action = handlers[handlers.length - 1] as Action;
    const refs = [...this.stack, ...(handlers.slice(0, -1) as MiddlewareRef[])];
    const full = join(this.prefix, path);
    (this.app.on as Function).call(this.app, methods, full, ...refs.map(resolveMiddleware), action);
    for (const method of methods) {
      routeTable.push({
        method,
        path: full,
        middleware: refs.map((r) => (typeof r === "string" ? r : r.name || "closure")),
        action: name ?? (action.name || "closure"),
      });
    }
    return this;
  }
}

/** Define a routes file. */
export const routes = (define: (r: Router) => void) => define;
