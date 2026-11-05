import { env } from "../core/env";

export default {
  // "sync" runs jobs inline (great for dev/tests); "redis" uses BullMQ workers.
  connection: env("QUEUE_CONNECTION", "sync") as "sync" | "redis",
  redis: env("REDIS_URL", "redis://localhost:6379"),
};
