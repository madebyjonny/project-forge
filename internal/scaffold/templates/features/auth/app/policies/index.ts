import type { Policy } from "~/core/policy";
import before from "./before";
// forge:imports

export default [
  before,
  // forge:policies
] satisfies Policy[] as Policy[];
