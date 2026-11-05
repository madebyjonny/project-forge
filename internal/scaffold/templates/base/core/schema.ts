import * as pg from "drizzle-orm/pg-core";
import * as sqlite from "drizzle-orm/sqlite-core";
import database from "../config/database";

/**
 * Dialect-agnostic column helpers. Models are written once; DB_CONNECTION decides
 * whether they become SQLite or Postgres tables. Types are expressed through the
 * SQLite builders so there's a single, consistent TypeScript surface.
 */
const isPg = () => database.connection === "postgres";

function pick<T>(lite: () => T, postgres: () => unknown): T {
  return isPg() ? (postgres() as T) : lite();
}

export const table = ((name: string, columns: any, extra?: any) =>
  isPg() ? pg.pgTable(name, columns, extra) : sqlite.sqliteTable(name, columns, extra)) as typeof sqlite.sqliteTable;

export const t = {
  /** UUID primary key. */
  id: (name = "id") =>
    pick(() => sqlite.text(name), () => pg.text(name)).primaryKey().$defaultFn(() => crypto.randomUUID()),

  string: (name: string) => pick(() => sqlite.text(name), () => pg.text(name)),
  text: (name: string) => pick(() => sqlite.text(name), () => pg.text(name)),
  integer: (name: string) => pick(() => sqlite.integer(name), () => pg.integer(name)),
  float: (name: string) => pick(() => sqlite.real(name), () => pg.doublePrecision(name)),
  boolean: (name: string) => pick(() => sqlite.integer(name, { mode: "boolean" }), () => pg.boolean(name)),
  timestamp: (name: string) =>
    pick(() => sqlite.integer(name, { mode: "timestamp_ms" }), () => pg.timestamp(name, { mode: "date", withTimezone: true })),
  json: <T = unknown>(name: string) =>
    pick(() => sqlite.text(name, { mode: "json" }), () => pg.jsonb(name)).$type<T>(),

  /** Foreign key to another table's id, cascading on delete. */
  foreignId: (name: string, references: () => sqlite.AnySQLiteColumn) =>
    pick(() => sqlite.text(name), () => pg.text(name)).references(references, { onDelete: "cascade" }),

  /**
   * Columns for tables owned by a library that serialises values itself (e.g. the
   * auth layer): JSON-in-text on SQLite, native text[] / jsonb on Postgres.
   */
  raw: {
    stringArray: (name: string) => pick(() => sqlite.text(name), () => pg.text(name).array()),
    json: (name: string) => pick(() => sqlite.text(name), () => pg.jsonb(name)),
  },

  /** created_at / updated_at, maintained automatically. */
  timestamps: () => ({
    createdAt: t.timestamp("created_at").notNull().$defaultFn(() => new Date()),
    updatedAt: t
      .timestamp("updated_at")
      .notNull()
      .$defaultFn(() => new Date())
      .$onUpdate(() => new Date()),
  }),
};

export { relations } from "drizzle-orm";
