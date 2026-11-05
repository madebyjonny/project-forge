// WebSockets + channel authorization (≈ Broadcasting + routes/channels.php).
// Clients send {"type":"subscribe","channel":"post.123"}; access is decided by
// routes/channels.ts and denied by default.
import { upgradeWebSocket, websocket } from "hono/bun";
import type { WSContext } from "hono/ws";
import { z } from "zod";
import config from "../config/broadcasting";
import channelRoutes from "../routes/channels";
import type { Feature } from "./app";
import { user, type AuthUser } from "./http";

export type ChannelAuth = (user: AuthUser | null, params: Record<string, string>) => boolean | Promise<boolean>;

/** Define channel authorization callbacks: { "post.{id}": (user, { id }) => ... } */
export function channels(defs: Record<string, ChannelAuth>) {
  return Object.entries(defs).map(([pattern, check]) => {
    const keys: string[] = [];
    const source = pattern
      .replace(/[.*+?^$()|[\]\\]/g, "\\$&")
      .replace(/\\?\{(\w+)\\?\}/g, (_m, key) => (keys.push(key), "([^.]+)"));
    return { pattern: new RegExp(`^${source}$`), keys, check };
  });
}

async function canJoin(u: AuthUser | null, channel: string) {
  for (const { pattern, keys, check } of channelRoutes) {
    const m = channel.match(pattern);
    if (m) return check(u, Object.fromEntries(keys.map((k, i) => [k, m[i + 1]!])));
  }
  return false;
}

const subscribers = new Map<string, Set<WSContext>>();

function deliver(channel: string, payload: string) {
  for (const ws of subscribers.get(channel) ?? []) ws.send(payload);
}

let redisPub: any;
const load = (pkg: string): Promise<any> => import(pkg);

/** Send an event to everyone on a channel (≈ broadcast(new Event)). */
export async function broadcast(channel: string, event: string, data: unknown = {}) {
  const payload = JSON.stringify({ type: "event", channel, event, data });
  if (config.connection === "redis") {
    const { default: Redis } = await load("ioredis");
    redisPub ??= new Redis(config.redis);
    await redisPub.publish(`broadcast:${channel}`, payload);
  } else {
    deliver(channel, payload);
  }
}

async function relayFromRedis() {
  const { default: Redis } = await load("ioredis");
  const sub = new Redis(config.redis);
  await sub.psubscribe("broadcast:*");
  sub.on("pmessage", (_p: string, ch: string, payload: string) => deliver(ch.slice("broadcast:".length), payload));
}

const Inbound = z.discriminatedUnion("type", [
  z.object({ type: z.literal("subscribe"), channel: z.string().max(200) }),
  z.object({ type: z.literal("unsubscribe"), channel: z.string().max(200) }),
  z.object({ type: z.literal("ping") }),
]);

export default function realtimeFeature(): Feature {
  return {
    name: "realtime",
    serve: { websocket },
    setup(app) {
      if (config.connection === "redis") void relayFromRedis();

      app.get(
        config.path,
        upgradeWebSocket((c) => {
          // Identity comes from the session resolved during the HTTP upgrade — never from messages.
          const u = user(c as any);
          const joined = new Set<string>();
          let window = Date.now();
          let count = 0;
          const send = (ws: WSContext, msg: object) => ws.send(JSON.stringify(msg));

          return {
            onOpen: (_e, ws) => send(ws, { type: "ready", user: u?.id ?? null }),
            async onMessage(evt, ws) {
              if (Date.now() - window > 1000) (window = Date.now()), (count = 0);
              if (++count > config.rateLimit) return ws.close(4429, "Too many messages");

              let raw: unknown;
              try {
                raw = JSON.parse(String(evt.data));
              } catch {
                return send(ws, { type: "error", reason: "bad_message" });
              }
              const parsed = Inbound.safeParse(raw);
              if (!parsed.success) return send(ws, { type: "error", reason: "bad_message" });
              const msg = parsed.data;

              if (msg.type === "ping") return send(ws, { type: "pong" });
              if (msg.type === "subscribe") {
                if (!(await canJoin(u, msg.channel))) return send(ws, { type: "denied", channel: msg.channel });
                joined.add(msg.channel);
                (subscribers.get(msg.channel) ?? subscribers.set(msg.channel, new Set()).get(msg.channel)!).add(ws);
                return send(ws, { type: "subscribed", channel: msg.channel });
              }
              joined.delete(msg.channel);
              subscribers.get(msg.channel)?.delete(ws);
              send(ws, { type: "unsubscribed", channel: msg.channel });
            },
            onClose(_e, ws) {
              for (const ch of joined) subscribers.get(ch)?.delete(ws);
            },
          };
        }),
      );
    },
  };
}
