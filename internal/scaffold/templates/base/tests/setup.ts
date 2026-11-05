// Preloaded by `bun test` (see bunfig.toml). Tests use an in-memory SQLite DB
// (.env.test), migrated fresh for each run — ≈ Laravel's RefreshDatabase.
import { migrate } from "~/core/migrator";

await migrate();
