import { controller } from "~/core/http";
import config from "~/config/app";

export default controller({
  index: (c) => c.json({ message: `Hello from ${config.name}!` }),

  hello: (c) => c.json({ message: `Hello, ${c.req.param("name") ?? "world"}!` }),

  health: (c) => c.json({ ok: true, uptime: process.uptime() }),
});
