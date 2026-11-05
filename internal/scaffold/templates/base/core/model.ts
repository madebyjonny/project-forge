import { and, count, eq, type SQL } from "drizzle-orm";
import type { SQLiteColumn, SQLiteTable } from "drizzle-orm/sqlite-core";
import { db } from "./db";
import { NotFoundError } from "./errors";

/** Non-enumerable tag on every row so policies know which model a record belongs to. */
export const MODEL = Symbol.for("forge.model");

type Table = SQLiteTable & { id: SQLiteColumn };

export interface Page<T> {
  data: T[];
  meta: { page: number; perPage: number; total: number; lastPage: number };
}

/**
 * Eloquent-flavoured helpers over a Drizzle table. Reach for `Model.query()` or
 * `db` directly whenever you need joins, ordering, or anything more advanced.
 */
export function model<T extends Table>(name: string, table: T) {
  type Row = T["$inferSelect"];
  type NewRow = T["$inferInsert"];

  // Resolved per call: models and the db module import each other.
  const q = () => db as any;
  const tag = (row: any): Row => {
    if (row && typeof row === "object") Object.defineProperty(row, MODEL, { value: name, enumerable: false });
    return row;
  };
  const whereOf = (attrs: Partial<Row>): SQL | undefined =>
    and(...Object.entries(attrs).map(([key, value]) => eq((table as any)[key], value)));

  return {
    name,
    table,

    /** A Drizzle select builder for this table. */
    query: () => q().select().from(table),

    async all(): Promise<Row[]> {
      return (await q().select().from(table)).map(tag);
    },

    async find(id: string): Promise<Row | undefined> {
      const [row] = await q().select().from(table).where(eq(table.id, id)).limit(1);
      return row ? tag(row) : undefined;
    },

    async findOrFail(id: string): Promise<Row> {
      const row = await this.find(id);
      if (!row) throw new NotFoundError(`${name} not found`);
      return row;
    },

    async where(attrs: Partial<Row>): Promise<Row[]> {
      return (await q().select().from(table).where(whereOf(attrs))).map(tag);
    },

    async first(attrs: Partial<Row>): Promise<Row | undefined> {
      const [row] = await q().select().from(table).where(whereOf(attrs)).limit(1);
      return row ? tag(row) : undefined;
    },

    async create(data: NewRow): Promise<Row> {
      const [row] = await q().insert(table).values(data).returning();
      return tag(row);
    },

    async update(id: string, data: Partial<NewRow>): Promise<Row> {
      const [row] = await q().update(table).set(data).where(eq(table.id, id)).returning();
      if (!row) throw new NotFoundError(`${name} not found`);
      return tag(row);
    },

    async delete(id: string): Promise<boolean> {
      const rows = await q().delete(table).where(eq(table.id, id)).returning();
      return rows.length > 0;
    },

    async count(attrs: Partial<Row> = {}): Promise<number> {
      const [row] = await q().select({ n: count() }).from(table).where(whereOf(attrs));
      return Number(row?.n ?? 0);
    },

    async paginate(page = 1, perPage = 15, attrs: Partial<Row> = {}): Promise<Page<Row>> {
      page = Math.max(1, Math.floor(page) || 1);
      perPage = Math.min(100, Math.max(1, Math.floor(perPage) || 15));
      const [data, total] = await Promise.all([
        q().select().from(table).where(whereOf(attrs)).limit(perPage).offset((page - 1) * perPage),
        this.count(attrs),
      ]);
      return { data: data.map(tag), meta: { page, perPage, total, lastPage: Math.max(1, Math.ceil(total / perPage)) } };
    },
  };
}
