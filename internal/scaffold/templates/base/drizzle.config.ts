import { defineConfig } from "drizzle-kit";
import database from "./config/database";

const postgres = database.connection === "postgres";

export default defineConfig({
  dialect: postgres ? "postgresql" : "sqlite",
  schema: "./app/models/*.ts",
  out: `./database/migrations/${database.connection}`,
  dbCredentials: { url: postgres ? database.postgres.url : database.sqlite.database },
});
