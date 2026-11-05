import type { Job } from "~/core/queue";
// forge:imports

// Jobs registered here are picked up by `forge queue:work`.
export default [
  // forge:jobs
] satisfies Job[] as Job[];
