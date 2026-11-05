import { env } from "../core/env";

export type Connection = "sqlite" | "postgres";

export default {
  // Switch per environment in .env / .env.production — no code changes needed.
  connection: env("DB_CONNECTION", "sqlite") as Connection,

  sqlite: {
    database: env("DB_DATABASE", "database/database.sqlite"),
  },

  postgres: {
    url: env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/app"),
  },
};
