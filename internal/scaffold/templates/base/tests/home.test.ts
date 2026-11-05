import { describe, expect, it } from "bun:test";
import { app } from "~/bootstrap/app";

describe("home", () => {
  it("says hello", async () => {
    const res = await app.request("/");
    expect(res.status).toBe(200);
    expect(await res.json()).toHaveProperty("message");
  });

  it("greets by name", async () => {
    const res = await app.request("/api/hello/forge");
    expect(await res.json()).toEqual({ message: "Hello, forge!" });
  });
});
