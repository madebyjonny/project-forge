/**
 * Read an environment variable with a typed fallback (≈ Laravel's env()).
 * Bun loads .env, .env.{NODE_ENV} and .env.local automatically.
 */
export function env(key: string): string | undefined;
export function env(key: string, fallback: string): string;
export function env(key: string, fallback: number): number;
export function env(key: string, fallback: boolean): boolean;
export function env(key: string, fallback?: string | number | boolean) {
  const raw = process.env[key];
  if (raw === undefined || raw === "") return fallback;
  if (typeof fallback === "number") return Number(raw);
  if (typeof fallback === "boolean") return ["1", "true", "yes", "on"].includes(raw.toLowerCase());
  return raw;
}

export const environment = () => env("NODE_ENV", "development");
export const isProduction = () => environment() === "production";
export const isTesting = () => environment() === "test";
