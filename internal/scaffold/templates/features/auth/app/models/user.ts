import { model } from "../../core/model";
import { t, table } from "../../core/schema";

export const users = table("users", {
  id: t.id(),
  name: t.string("name").notNull(),
  email: t.string("email").notNull().unique(),
  emailVerified: t.boolean("email_verified").notNull().default(false),
  image: t.string("image"),
  role: t.string("role").notNull().default("member"),
  ...t.timestamps(),
});

export const User = model("User", users);
export type User = typeof users.$inferSelect;
