// Tables owned by the auth layer (sessions, OAuth accounts, verification tokens).
import { t, table } from "../../core/schema";
import { users } from "./user";

export const sessions = table("sessions", {
  id: t.id(),
  userId: t.foreignId("user_id", () => users.id).notNull(),
  token: t.string("token").notNull().unique(),
  expiresAt: t.timestamp("expires_at").notNull(),
  ipAddress: t.string("ip_address"),
  userAgent: t.string("user_agent"),
  ...t.timestamps(),
});

export const accounts = table("accounts", {
  id: t.id(),
  userId: t.foreignId("user_id", () => users.id).notNull(),
  accountId: t.string("account_id").notNull(),
  providerId: t.string("provider_id").notNull(),
  accessToken: t.text("access_token"),
  refreshToken: t.text("refresh_token"),
  idToken: t.text("id_token"),
  accessTokenExpiresAt: t.timestamp("access_token_expires_at"),
  refreshTokenExpiresAt: t.timestamp("refresh_token_expires_at"),
  scope: t.string("scope"),
  password: t.string("password"),
  ...t.timestamps(),
});

export const verifications = table("verifications", {
  id: t.id(),
  identifier: t.string("identifier").notNull(),
  value: t.text("value").notNull(),
  expiresAt: t.timestamp("expires_at").notNull(),
  ...t.timestamps(),
});
