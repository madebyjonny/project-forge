import { policy } from "~/core/policy";

// Runs first — admins can do anything (≈ Gate::before).
export default policy((user, { can }) => {
  if (user?.role === "admin") can("manage", "all");
});
