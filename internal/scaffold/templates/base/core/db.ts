import { mkdirSync } from "node:fs";
import { dirname } from "node:path";
import { Database } from "bun:sqlite";
import { drizzle as drizzleSqlite, type BunSQLiteDatabase } from "drizzle-orm/bun-sqlite";
import { drizzle as drizzlePg } from "drizzle-orm/node-postgres";
import { Pool } from "pg";
import database from "../config/database";
import * as schema from "../app/models";

export type DB = BunSQLiteDatabase<typeof schema>;

let instance: DB | undefined;
let close: (() => Promise<void> | void) | undefined;

/** Lazily open the configured connection (sqlite or postgres). */
export function connect(): DB {
  if (instance) return instance;

  if (database.connection === "postgres") {
    const pool = new Pool({ connectionString: database.postgres.url });
    instance = drizzlePg(pool, { schema }) as unknown as DB;
    close = () => pool.end();
  } else {
    const file = database.sqlite.database;
    if (file !== ":memory:") mkdirSync(dirname(file), { recursive: true });
    const client = new Database(file, { create: true });
    client.exec("PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON;");
    instance = drizzleSqlite(client, { schema });
    close = () => client.close();
  }
  return instance;
}

export async function disconnect() {
  await close?.();
  instance = undefined;
  close = undefined;
}

/** The query builder — full Drizzle power when the model helpers aren't enough. */
export const db: DB = new Proxy({} as DB, {
  get(_target, prop) {
    const conn = connect() as any;
    const value = conn[prop];
    return typeof value === "function" ? value.bind(conn) : value;
  },
});
