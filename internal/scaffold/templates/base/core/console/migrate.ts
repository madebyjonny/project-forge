// Invoked by `forge migrate` via the `forge:migrate` package script.
import database from "../../config/database";
import { disconnect } from "../db";
import { migrate, wipe } from "../migrator";

const args = process.argv.slice(2);

try {
  if (args.includes("--fresh")) {
    await wipe();
    console.log(`Dropped all tables (${database.connection}).`);
  }
  const ran = await migrate();
  console.log(ran ? `Migrated (${database.connection}).` : "Nothing to migrate.");

  if (args.includes("--seed")) {
    const { default: seed } = await import("../../database/seeders");
    await seed();
    console.log("Seeded.");
  }
} finally {
  await disconnect();
}
