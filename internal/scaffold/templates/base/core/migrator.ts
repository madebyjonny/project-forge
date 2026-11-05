import { existsSync, rmSync } from "node:fs";
import { sql } from "drizzle-orm";
import { migrate as migrateSqlite } from "drizzle-orm/bun-sqlite/migrator";
import { migrate as migratePg } from "drizzle-orm/node-postgres/migrator";
import database from "../config/database";
import { connect, disconnect } from "./db";

export const migrationsFolder = () => `database/migrations/${database.connection}`;

/** Apply pending migrations for the active connection. Returns false if there are none. */
export async function migrate(): Promise<boolean> {
  const folder = migrationsFolder();
  if (!existsSync(`${folder}/meta/_journal.json`)) return false;
  const db = connect() as any;
  if (database.connection === "postgres") await migratePg(db, { migrationsFolder: folder });
  else migrateSqlite(db, { migrationsFolder: folder });
  return true;
}

/** Drop everything and start over (≈ migrate:fresh). */
export async function wipe() {
  if (database.connection === "postgres") {
    const db = connect() as any;
    await db.execute(sql`DROP SCHEMA IF EXISTS drizzle CASCADE`);
    await db.execute(sql`DROP SCHEMA public CASCADE`);
    await db.execute(sql`CREATE SCHEMA public`);
    return;
  }
  await disconnect();
  const file = database.sqlite.database;
  if (file === ":memory:") return;
  for (const suffix of ["", "-wal", "-shm"]) rmSync(file + suffix, { force: true });
}
