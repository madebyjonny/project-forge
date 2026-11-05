import type { Middleware } from "~/core/middleware";
// forge:imports

// Named middleware, usable in routes as strings: r.get("/x", "name", action)
// Built in: "auth", "guest", "role:admin", "throttle:60,60" (+ "can:update,Post" with auth).
export default {
  // forge:middleware
} satisfies Record<string, Middleware>;
