// Invoked by `forge db:seed`.
import { disconnect } from "../db";
import seed from "../../database/seeders";

try {
  await seed();
  console.log("Seeded.");
} finally {
  await disconnect();
}
