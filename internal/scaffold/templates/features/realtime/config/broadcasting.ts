import { env } from "../core/env";

export default {
  // "local" = this process only; "redis" fans out across every server instance.
  connection: env("BROADCAST_CONNECTION", "local") as "local" | "redis",
  redis: env("REDIS_URL", "redis://localhost:6379"),
  path: "/ws",
  // Max inbound messages per socket per second before it's closed.
  rateLimit: 20,
};
