import { describe, expect, it } from "bun:test";
import { app } from "~/bootstrap/app";

const json = (body: unknown, headers: Record<string, string> = {}) => ({
  method: "POST",
  headers: { "content-type": "application/json", origin: "http://localhost:3000", ...headers },
  body: JSON.stringify(body),
});

describe("auth", () => {
  it("rejects guests", async () => {
    const res = await app.request("/api/me");
    expect(res.status).toBe(401);
  });

  it("signs up and reads the current user", async () => {
    const res = await app.request(
      "/api/auth/sign-up/email",
      json({ name: "Ada", email: "ada@example.com", password: "correct-horse-battery" }),
    );
    expect(res.status).toBe(200);
    const cookie = res.headers.get("set-cookie")!;

    const me = await app.request("/api/me", { headers: { cookie } });
    expect(me.status).toBe(200);
    const body = (await me.json()) as { user: { email: string } };
    expect(body.user.email).toBe("ada@example.com");
  });
});
