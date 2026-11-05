// Invoked by `forge make:migration`. Generates SQL for every dialect so you can
// switch DB_CONNECTION per environment without regenerating anything.
const args = process.argv.slice(2);
const nameIdx = args.indexOf("--name");
const name = nameIdx >= 0 ? args[nameIdx + 1] : undefined;

for (const connection of ["sqlite", "postgres"]) {
  console.log(`\n▸ ${connection}`);
  const proc = Bun.spawnSync(["bun", "--bun", "x", "drizzle-kit", "generate", ...(name ? ["--name", name] : [])], {
    env: { ...process.env, DB_CONNECTION: connection },
    stdout: "inherit",
    stderr: "inherit",
  });
  if (proc.exitCode !== 0) process.exit(proc.exitCode ?? 1);
}
