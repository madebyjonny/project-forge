// Invoked by `forge route:list`. Prints the route table as JSON for the CLI to render.
import { app } from "../../bootstrap/app";
import { routeTable } from "../router";

void app;
// Routes added by features (auth, mcp, …). Catch-all "ALL" entries are global middleware.
const builtins = (app.routes as { method: string; path: string }[])
  .filter((r) => !(r.method === "ALL" && r.path.includes("*")))
  .map((r) => ({ method: r.method === "ALL" ? "ANY" : r.method, path: r.path }))
  .filter((r) => !routeTable.some((t) => t.path === r.path && t.method === r.method))
  .filter((r, i, all) => all.findIndex((o) => o.method === r.method && o.path === r.path) === i)
  .map((r) => ({ ...r, middleware: [], action: "feature" }));

console.log(JSON.stringify([...routeTable, ...builtins]));
process.exit(0);
