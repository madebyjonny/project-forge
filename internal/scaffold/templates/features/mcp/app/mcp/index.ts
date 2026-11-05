import type { Tool } from "~/core/mcp";
import whoami from "./tools/whoami";
// forge:imports

// Tools exposed at /mcp. `forge make:tool` keeps this list up to date.
export default [
  whoami,
  // forge:tools
] satisfies Tool[] as Tool[];
